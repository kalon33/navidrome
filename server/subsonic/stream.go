package subsonic

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/capabilities"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/httpclient"
	"github.com/navidrome/navidrome/utils/req"
)

// podcastEnclosureUserAgent is sent when proxying a podcast enclosure. Podcast
// hosting CDNs commonly block or throttle unknown user agents (returning an
// HTML error page instead of the audio), which breaks playback in Subsonic
// clients streaming through /rest/stream while the web UI (using the browser
// user agent) keeps working. This generic, widely accepted user agent makes the
// publishers serve the real audio bytes to the proxy.
const podcastEnclosureUserAgent = "Mozilla/5.0 (compatible; Navidrome Podcast Proxy)"

func (api *Router) Stream(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	// Podcast episodes live in the plugin's KVStore, not in the media library.
	// Their IDs are prefixed "ep-" and stream directly from the publisher's
	// enclosure URL, so they are proxied instead of going through the streamer.
	if isPodcastEpisodeID(id) {
		return api.streamPodcastEpisode(w, r, stripTranscodeSuffix(id))
	}

	maxBitRate := p.IntOr("maxBitRate", 0)
	format, _ := p.String("format")
	timeOffset := p.IntOr("timeOffset", 0)

	mf, err := api.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	streamReq := api.transcodeDecision.ResolveRequest(ctx, mf, format, maxBitRate, timeOffset)
	stream, err := api.streamer.NewStream(ctx, mf, streamReq)
	if err != nil {
		return nil, err
	}

	// Make sure the stream will be closed at the end, to avoid leakage
	defer func() {
		if err := stream.Close(); err != nil && log.IsGreaterOrEqualTo(log.LevelDebug) {
			log.Error("Error closing stream", "id", id, "file", stream.Name(), err)
		}
	}()

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Content-Duration", strconv.FormatFloat(float64(stream.Duration()), 'G', -1, 32))

	_, err = stream.Serve(ctx, w, r)
	return nil, err
}

// isPodcastEpisodeID reports whether the given ID refers to a podcast episode
// managed by the podcast plugin (prefix "ep-"), rather than a media file in
// the library.
func isPodcastEpisodeID(id string) bool {
	return strings.HasPrefix(id, "ep-")
}

// streamPodcastEpisode proxies the podcast episode's enclosure URL so external
// Subsonic clients can play it through the standard stream endpoint. The
// request's Range header is forwarded to the publisher to keep episodes
// seekable, and the response's content type / length / range headers are
// passed through unchanged.
func (api *Router) streamPodcastEpisode(w http.ResponseWriter, r *http.Request, id string) (*responses.Subsonic, error) {
	return api.proxyPodcastEpisode(w, r, id, false)
}

// downloadPodcastEpisode proxies the podcast episode's enclosure URL with an
// attachment disposition so Subsonic clients can save it via /rest/download.
func (api *Router) downloadPodcastEpisode(w http.ResponseWriter, r *http.Request, id string) (*responses.Subsonic, error) {
	return api.proxyPodcastEpisode(w, r, id, true)
}

// proxyPodcastEpisode fetches the podcast episode identified by id from the
// plugin and streams its enclosure URL back to the client. When asDownload is
// set, a Content-Disposition: attachment header forces the client to save the
// file rather than play it. Range/Accept headers are forwarded so streamed
// episodes stay seekable, and content headers from the publisher are relayed.
func (api *Router) proxyPodcastEpisode(w http.ResponseWriter, r *http.Request, id string, asDownload bool) (*responses.Subsonic, error) {
	ctx := r.Context()
	if api.podcast == nil || !api.podcast.HasProvider() {
		return nil, errNoPodcastProvider()
	}
	episode, err := api.podcast.GetEpisode(ctx, id)
	if err != nil {
		log.Error(ctx, "Error retrieving podcast episode for stream", "id", id, err)
		return nil, newError(responses.ErrorDataNotFound, "podcast episode not found: %s", id)
	}
	if episode == nil || episode.StreamURL == "" {
		log.Warn(ctx, "Podcast episode not streamable (nil or no stream URL)", "id", id)
		return nil, newError(responses.ErrorDataNotFound, "podcast episode not streamable: %s", id)
	}
	log.Debug(ctx, "Proxying podcast episode", "id", id, "url", episode.StreamURL, "contentType", episode.ContentType, "duration", episode.Duration, "size", episode.Size)

	proxyReq, err := http.NewRequestWithContext(ctx, http.MethodGet, episode.StreamURL, nil) //nolint:gosec // StreamURL is a publisher enclosure from a podcast feed the user subscribed to
	if err != nil {
		log.Error(ctx, "Error building podcast stream request", "id", id, "url", episode.StreamURL, err)
		return nil, newError(responses.ErrorGeneric, "error streaming podcast episode")
	}
	// Forward Range and Accept headers so the publisher can serve partial
	// content and keep the episode seekable in clients.
	for _, h := range []string{"Range", "Accept", "If-Range"} {
		if v := r.Header.Get(h); v != "" {
			proxyReq.Header.Set(h, v)
		}
	}
	// Request the enclosure byte-for-byte. Go's default Transport adds
	// "Accept-Encoding: gzip" and transparently decompresses gzipped responses,
	// which strips Content-Length from the response and breaks seeking/progress
	// for clients that rely on it. Asking for "identity" makes the publisher
	// serve the raw bytes with a usable Content-Length, and also keeps Range
	// requests meaningful (auto-decoding a partial gzipped document fails).
	proxyReq.Header.Set("Accept-Encoding", "identity")
	// Podcast enclosures are served by hosting CDNs (e.g. Ausha, Radio France)
	// that routinely block or throttle requests from unknown user agents, often
	// returning an HTML error page with a 2xx/4xx status. ExoPlayer then tries to
	// extract that error body as audio and fails silently, while the web UI works
	// because the browser fetches the enclosure with its own user agent. Use a
	// generic, widely accepted podcast user agent so publishers serve the real
	// audio bytes to the proxy.
	proxyReq.Header.Set("User-Agent", podcastEnclosureUserAgent)

	client := httpclient.New(0)
	resp, err := client.Do(proxyReq) //nolint:gosec // proxyReq targets the user-subscribed podcast enclosure
	if err != nil {
		log.Error(ctx, "Error fetching podcast enclosure", "id", id, "url", episode.StreamURL, err)
		return nil, newError(responses.ErrorGeneric, "error streaming podcast episode")
	}
	defer resp.Body.Close()
	log.Debug(ctx, "Podcast enclosure response", "id", id, "status", resp.StatusCode, "contentType", resp.Header.Get("Content-Type"), "contentEncoding", resp.Header.Get("Content-Encoding"), "contentLength", resp.Header.Get("Content-Length"), "acceptRanges", resp.Header.Get("Accept-Ranges"), "contentRange", resp.Header.Get("Content-Range"))
	// A non-2xx response from the publisher (forbidden, not found, ...) means no
	// audio is available. Relaying it verbatim lets clients try to decode an HTML
	// error page as audio and fail silently, so log it loudly instead.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Warn(ctx, "Podcast enclosure returned an error status", "id", id, "url", episode.StreamURL, "status", resp.StatusCode)
		return nil, newError(responses.ErrorDataNotFound, "podcast enclosure unavailable (HTTP %d)", resp.StatusCode)
	}
	// We ask for Accept-Encoding: identity, but some CDNs gzip the response
	// regardless. Go only transparently decompresses gzip it requested itself, so
	// here resp.Body is the still-compressed bytes with a Content-Length matching
	// the gzipped size. Relaying that verbatim would send compressed audio
	// (without a Content-Encoding header) that ExoPlayer cannot decode, causing
	// silent playback failures while the web UI (browser auto-decodes) works.
	// Detect this case and decompress server-side.
	body := resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, gzErr := gzip.NewReader(resp.Body)
		if gzErr != nil {
			log.Warn(ctx, "Could not open gzip reader for podcast enclosure", "id", id, "url", episode.StreamURL, gzErr)
			return nil, newError(responses.ErrorGeneric, "error streaming podcast episode")
		}
		body = &gzipCloseReader{gz: gz, body: resp.Body}
	}
	defer body.Close()
	// Relay status code and content-related headers from the publisher. The
	// Content-Length is dropped when decompressing gzip, since it described the
	// compressed size, not the audio size; a wrong length would break seeking.
	relayHeaders := []string{"Content-Type", "Content-Range", "Accept-Ranges"}
	if !strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		relayHeaders = append([]string{"Content-Length"}, relayHeaders...)
	}
	for _, h := range relayHeaders {
		if v := resp.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	// Some publishers omit Content-Type; fall back to the episode's declared type
	// (or one derived from its suffix) so clients can pick the right extractor.
	if w.Header().Get("Content-Type") == "" && episode.ContentType != "" {
		w.Header().Set("Content-Type", episode.ContentType)
	}
	if asDownload {
		name := downloadFilename(episode)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if episode.Duration > 0 {
		w.Header().Set("X-Content-Duration", strconv.FormatFloat(float64(episode.Duration), 'G', -1, 32))
	}

	w.WriteHeader(resp.StatusCode)
	if r.Method == http.MethodHead {
		return nil, nil
	}
	if _, err := io.Copy(w, body); err != nil {
		// A broken pipe / connection reset just means the client went away
		// (track change, seek, stop): it is expected, not a server fault.
		if isClientDisconnect(err) {
			log.Debug(ctx, "Podcast episode proxy: client disconnected", "id", id, err)
		} else {
			log.Warn(ctx, "Error proxying podcast episode", "id", id, err)
		}
	}
	return nil, nil
}

// isClientDisconnect reports whether err is a client-side network error (the
// client closed the connection while we were still writing): a broken pipe,
// a connection reset, or a canceled request context. These are expected during
// normal playback (track change, seek, stop) and should not be logged as
// warnings.
func isClientDisconnect(err error) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "EOF")
}

// gzipCloseReader wraps a gzip.Reader so closing it also closes the underlying
// response body, and so it satisfies io.ReadCloser for the proxy copy.
type gzipCloseReader struct {
	gz   *gzip.Reader
	body io.Closer
}

func (g *gzipCloseReader) Read(p []byte) (int, error) { return g.gz.Read(p) }
func (g *gzipCloseReader) Close() error {
	_ = g.gz.Close()
	return g.body.Close()
}

// downloadFilename builds a safe attachment filename for a podcast episode,
// using its suffix when known and falling back to the episode id.
func downloadFilename(ep *capabilities.PodcastEpisode) string {
	base := ep.Title
	if base == "" {
		base = ep.ID
	}
	base = strings.ReplaceAll(base, "/", "_")
	if ep.Suffix != "" {
		return base + "." + ep.Suffix
	}
	return base
}

func (api *Router) Download(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	username, _ := request.UsernameFrom(ctx)
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}

	if !conf.Server.EnableDownloads {
		log.Warn(ctx, "Downloads are disabled", "user", username, "id", id)
		return nil, newError(responses.ErrorAuthorizationFail, "downloads are disabled")
	}

	// Podcast episodes are not library media files; proxy their enclosure so
	// clients can download episodes via the standard download endpoint.
	if isPodcastEpisodeID(id) {
		return api.downloadPodcastEpisode(w, r, stripTranscodeSuffix(id))
	}

	entity, err := model.GetEntityByID(ctx, api.ds, id)
	if err != nil {
		return nil, err
	}

	maxBitRate := p.IntOr("bitrate", 0)
	format, _ := p.String("format")

	if format == "" {
		if conf.Server.AutoTranscodeDownload {
			// if we are not provided a format, see if we have requested transcoding for this client
			// This must be enabled via a config option. For the UI, we are always given an option.
			// This will impact other clients which do not use the UI
			transcoding, ok := request.TranscodingFrom(ctx)

			if !ok {
				format = "raw"
			} else {
				format = transcoding.TargetFormat
				maxBitRate = transcoding.DefaultBitRate
			}
		} else {
			format = "raw"
		}
	}

	setHeaders := func(name string) {
		name = strings.ReplaceAll(name, ",", "_")
		disposition := fmt.Sprintf("attachment; filename=\"%s.zip\"", name)
		w.Header().Set("Content-Disposition", disposition)
		w.Header().Set("Content-Type", "application/zip")
	}

	switch v := entity.(type) {
	case *model.MediaFile:
		streamReq := api.transcodeDecision.ResolveRequest(ctx, v, format, maxBitRate, 0)
		stream, err := api.streamer.NewStream(ctx, v, streamReq)
		if err != nil {
			return nil, err
		}

		// Make sure the stream will be closed at the end, to avoid leakage
		defer func() {
			if err := stream.Close(); err != nil && log.IsGreaterOrEqualTo(log.LevelDebug) {
				log.Error("Error closing stream", "id", id, "file", stream.Name(), err)
			}
		}()

		disposition := fmt.Sprintf("attachment; filename=\"%s\"", stream.Name())
		w.Header().Set("Content-Disposition", disposition)

		_, err = stream.Serve(ctx, w, r)
		return nil, err
	case *model.Album:
		setHeaders(v.Name)
		return nil, handleArchiveErr(ctx, id, api.archiver.ZipAlbum(ctx, id, format, maxBitRate, w))
	case *model.Artist:
		setHeaders(v.Name)
		return nil, handleArchiveErr(ctx, id, api.archiver.ZipArtist(ctx, id, format, maxBitRate, w))
	case *model.Playlist:
		setHeaders(v.Name)
		return nil, handleArchiveErr(ctx, id, api.archiver.ZipPlaylist(ctx, id, format, maxBitRate, w))
	default:
		return nil, model.ErrNotFound
	}
}

// handleArchiveErr swallows ErrTooManyTranscodes from archive downloads so the
// outer error handler does not try to write a 429 onto a response whose status
// and Content-Disposition have already been flushed. The archive ends up with
// the tracks that were written before the rejection (the rejected track and
// any following ones are omitted); the server-side log is the unambiguous
// signal operators can act on.
func handleArchiveErr(ctx context.Context, id string, err error) error {
	if errors.Is(err, stream.ErrTooManyTranscodes) {
		log.Warn(ctx, "Archive download finalized early: transcode cap reached", "id", id, err)
		return nil
	}
	return err
}
