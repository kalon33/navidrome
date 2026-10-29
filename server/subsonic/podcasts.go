package subsonic

import (
	"context"
	"net/http"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/capabilities"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
)

// errNoPodcastProvider is returned when no podcast plugin is configured. It is
// mapped to a Subsonic generic error so clients receive a clear message instead
// of an empty result.
func errNoPodcastProvider() error {
	return newError(responses.ErrorGeneric, "Podcast support requires a podcast plugin to be configured")
}

// GetPodcasts returns all podcast channels, optionally with episodes.
func (api *Router) GetPodcasts(r *http.Request) (*responses.Subsonic, error) {
	if api.podcast == nil || !api.podcast.HasProvider() {
		return nil, errNoPodcastProvider()
	}
	ctx := r.Context()
	p := req.Params(r)
	includeEpisodes := p.BoolOr("includeEpisodes", true)
	id := p.StringOr("id", "")

	var channels []capabilities.PodcastChannel
	if id != "" {
		ch, e := api.podcast.GetChannel(ctx, id, includeEpisodes)
		if e != nil {
			return nil, e
		}
		if ch != nil {
			channels = []capabilities.PodcastChannel{*ch}
		}
	} else {
		c, e := api.podcast.GetChannels(ctx, includeEpisodes)
		if e != nil {
			return nil, e
		}
		channels = c
	}

	resp := make([]responses.PodcastChannel, len(channels))
	for i, ch := range channels {
		resp[i] = toPodcastChannel(ch)
	}
	response := newResponse()
	response.Podcasts = &responses.Podcasts{Channels: resp}
	return response, nil
}

// GetNewestPodcasts returns the most recently published podcast episodes.
func (api *Router) GetNewestPodcasts(r *http.Request) (*responses.Subsonic, error) {
	if api.podcast == nil || !api.podcast.HasProvider() {
		return nil, errNoPodcastProvider()
	}
	ctx := r.Context()
	p := req.Params(r)
	count := p.IntOr("count", 20)

	episodes, err := api.podcast.GetNewestEpisodes(ctx, count)
	if err != nil {
		return nil, err
	}

	// Resolve the feed names once so each episode carries its channel's title as
	// its artist, keeping the player display consistent with getPodcasts.
	channelTitles := api.podcastChannelTitles(ctx, episodes)

	resp := make([]responses.PodcastEpisode, len(episodes))
	for i, ep := range episodes {
		resp[i] = toPodcastEpisode(ep)
		title := channelTitles[ep.ChannelID]
		resp[i].Child.Artist = title
		resp[i].Child.Album = title
	}
	response := newResponse()
	response.NewestPodcasts = &responses.NewestPodcasts{Episodes: resp}
	return response, nil
}

// GetPodcastEpisode returns the metadata for a single podcast episode by ID.
func (api *Router) GetPodcastEpisode(r *http.Request) (*responses.Subsonic, error) {
	if api.podcast == nil || !api.podcast.HasProvider() {
		return nil, errNoPodcastProvider()
	}
	ctx := r.Context()
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	episode, err := api.podcast.GetEpisode(ctx, id)
	if err != nil {
		return nil, err
	}
	response := newResponse()
	response.PodcastEpisode = toPodcastEpisodePointer(episode)
	// Expose the feed name as the episode's artist, like getPodcasts does.
	if response.PodcastEpisode != nil {
		title := api.podcastChannelTitle(ctx, episode.ChannelID)
		response.PodcastEpisode.Child.Artist = title
		response.PodcastEpisode.Child.Album = title
	}
	return response, nil
}

// CreatePodcastChannel subscribes to a new podcast channel.
func (api *Router) CreatePodcastChannel(r *http.Request) (*responses.Subsonic, error) {
	if api.podcast == nil || !api.podcast.HasProvider() {
		return nil, errNoPodcastProvider()
	}
	ctx := r.Context()
	p := req.Params(r)
	url, err := p.String("url")
	if err != nil {
		return nil, err
	}
	if _, err := api.podcast.CreateChannel(ctx, url); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

// RefreshPodcasts requests the server to check for new podcast episodes.
func (api *Router) RefreshPodcasts(r *http.Request) (*responses.Subsonic, error) {
	if api.podcast == nil || !api.podcast.HasProvider() {
		return nil, errNoPodcastProvider()
	}
	ctx := r.Context()
	if _, err := api.podcast.RefreshChannels(ctx, nil); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

// DownloadPodcastEpisode requests the server to start downloading a podcast episode.
func (api *Router) DownloadPodcastEpisode(r *http.Request) (*responses.Subsonic, error) {
	if api.podcast == nil || !api.podcast.HasProvider() {
		return nil, errNoPodcastProvider()
	}
	ctx := r.Context()
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	if _, err := api.podcast.DownloadEpisode(ctx, id); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

// DeletePodcastChannel deletes a podcast channel.
func (api *Router) DeletePodcastChannel(r *http.Request) (*responses.Subsonic, error) {
	if api.podcast == nil || !api.podcast.HasProvider() {
		return nil, errNoPodcastProvider()
	}
	ctx := r.Context()
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	if err := api.podcast.DeleteChannel(ctx, id); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

// DeletePodcastEpisode deletes a podcast episode.
func (api *Router) DeletePodcastEpisode(r *http.Request) (*responses.Subsonic, error) {
	if api.podcast == nil || !api.podcast.HasProvider() {
		return nil, errNoPodcastProvider()
	}
	ctx := r.Context()
	p := req.Params(r)
	id, err := p.String("id")
	if err != nil {
		return nil, err
	}
	if err := api.podcast.DeleteEpisode(ctx, id); err != nil {
		return nil, err
	}
	return newResponse(), nil
}

func toPodcastChannel(ch capabilities.PodcastChannel) responses.PodcastChannel {
	resp := responses.PodcastChannel{
		Id:               ch.ID,
		Url:              ch.URL,
		Title:            ch.Title,
		Description:      ch.Description,
		CoverArt:         podcastCoverArt(ch.ID),
		OriginalImageUrl: ch.OriginalImageUrl,
		Status:           string(ch.Status),
		ErrorMessage:     ch.ErrorMessage,
	}
	if len(ch.Episodes) > 0 {
		resp.Episode = make([]responses.PodcastEpisode, len(ch.Episodes))
		for i, ep := range ch.Episodes {
			resp.Episode[i] = toPodcastEpisode(ep)
			// Expose the feed name as the episode's artist so players show it while
			// an episode plays (Subsonic Child artist/album attributes).
			resp.Episode[i].Child.Artist = ch.Title
			resp.Episode[i].Child.Album = ch.Title
		}
	}
	return resp
}

func toPodcastEpisode(ep capabilities.PodcastEpisode) responses.PodcastEpisode {
	resp := responses.PodcastEpisode{
		Child:       childFromPodcastEpisode(ep, ""),
		StreamId:    ep.StreamID,
		StreamUrl:   ep.StreamURL,
		ChannelId:   ep.ChannelID,
		Description: ep.Description,
		Status:      string(ep.Status),
		PublishDate: ep.PublishDate,
	}
	return resp
}

// childFromPodcastEpisode builds a Subsonic Child from a podcast episode, used both
// by getPodcasts/getNewestPodcasts and by getSong so external clients that fetch
// episode metadata via /rest/getSong receive a playable podcast entry. The
// channelTitle, when known, is exposed as the Child's artist so players display
// the podcast feed name while an episode is playing.
func childFromPodcastEpisode(ep capabilities.PodcastEpisode, channelTitle string) responses.Child {
	return responses.Child{
		Id:          ep.ID,
		Title:       ep.Title,
		IsDir:       false,
		Artist:      channelTitle,
		Album:       channelTitle,
		Year:        ep.Year,
		Genre:       ep.Genre,
		CoverArt:    episodeCoverArt(ep),
		Size:        ep.Size,
		ContentType: ep.ContentType,
		Suffix:      ep.Suffix,
		Duration:    ep.Duration,
		BitRate:     ep.BitRate,
		Path:        ep.Path,
		Type:        "podcast",
	}
}

func toPodcastEpisodePointer(ep *capabilities.PodcastEpisode) *responses.PodcastEpisode {
	if ep == nil {
		return nil
	}
	r := toPodcastEpisode(*ep)
	return &r
}

// podcastChannelTitle resolves the title (feed name) of a podcast channel by ID.
// It returns an empty string when the channel cannot be resolved; callers expose it
// as the episode's artist so players display the feed name during playback.
func (api *Router) podcastChannelTitle(ctx context.Context, channelID string) string {
	if api.podcast == nil || channelID == "" {
		return ""
	}
	ch, err := api.podcast.GetChannel(ctx, channelID, false)
	if err != nil || ch == nil {
		return ""
	}
	return ch.Title
}

// podcastChannelTitles resolves the titles of the channels the given episodes
// belong to, returning a channelID-to-title map. Channels that cannot be
// resolved are simply absent from the map.
func (api *Router) podcastChannelTitles(ctx context.Context, episodes []capabilities.PodcastEpisode) map[string]string {
	ids := make([]string, 0, len(episodes))
	seen := map[string]bool{}
	for _, ep := range episodes {
		if ep.ChannelID != "" && !seen[ep.ChannelID] {
			seen[ep.ChannelID] = true
			ids = append(ids, ep.ChannelID)
		}
	}
	titles := make(map[string]string, len(ids))
	for _, id := range ids {
		if t := api.podcastChannelTitle(ctx, id); t != "" {
			titles[id] = t
		}
	}
	return titles
}

// podcastCoverArt builds a resolvable artwork id for a podcast channel. Plugin coverArt
// ids (e.g. "ch-...") are opaque to the artwork service, so they are remapped to the
// podcast artwork kind (pc-), which resolves the channel's OriginalImageUrl remotely.
func podcastCoverArt(channelID string) string {
	if channelID == "" {
		return ""
	}
	return model.NewArtworkID(model.KindPodcastArtwork, channelID, nil).String()
}

// episodeCoverArt picks a usable cover id for an episode. Episodes rarely carry their
// own remote image, so they fall back to their channel's cover id (pc-<channelId>);
// an absolute http(s) cover is kept verbatim.
func episodeCoverArt(ep capabilities.PodcastEpisode) string {
	if ep.CoverArt != "" && (len(ep.CoverArt) > 7 && (ep.CoverArt[:7] == "http://" || ep.CoverArt[:8] == "https://")) {
		return ep.CoverArt
	}
	return podcastCoverArt(ep.ChannelID)
}
