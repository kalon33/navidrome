// Functional RSS podcast backend for Navidrome.
//
// Unlike a stub demo, this plugin is a real podcast backend: it subscribes to
// RSS feeds, fetches and parses them over HTTP, persists channels and episodes
// in the KVStore (surviving restarts), and schedules periodic feed refreshes
// via the Scheduler host service.
//
// It implements the full Podcast capability surface:
//   - getPodcasts         -> GetChannels
//   - getNewestPodcasts   -> GetNewestEpisodes
//   - getPodcastEpisode   -> GetEpisode
//   - createPodcastChannel-> CreateChannel
//   - refreshPodcasts      -> RefreshChannels
//   - downloadPodcastEpisode -> DownloadEpisode
//   - deletePodcastChannel -> DeleteChannel
//   - deletePodcastEpisode -> DeleteEpisode
//
// Episodes are not downloaded to disk: their StreamURL points at the enclosure
// (the media URL in the RSS feed), so Subsonic clients stream directly from the
// publisher. DownloadEpisode marks the episode as "completed" once the enclosure
// is reachable; a real backend would store the file via the Storage service.
//
// Build with:
//
//	tinygo build -o podcast.wasm -target wasip1 -buildmode=c-shared .
//	zip -j podcast.ndp manifest.json podcast.wasm
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/plugins/pdk/go/host"
	"github.com/navidrome/navidrome/plugins/pdk/go/lifecycle"
	"github.com/navidrome/navidrome/plugins/pdk/go/pdk"
	"github.com/navidrome/navidrome/plugins/pdk/go/podcast"
	"github.com/navidrome/navidrome/plugins/pdk/go/scheduler"
)

const (
	channelKeyPrefix = "channel:"
	channelURLPrefix = "channel-url:"
	indexChannels    = "index:channels"
	scheduleID       = "podcast-refresh"
	defaultUserAgent = "NavidromePodcastPlugin/1.0"
)

func init() {
	lifecycle.Register(&podcastPlugin{})
	scheduler.Register(&podcastPlugin{})
	podcast.Register(&podcastPlugin{})
}

type podcastPlugin struct{}

var (
	_ lifecycle.InitProvider            = (*podcastPlugin)(nil)
	_ scheduler.CallbackProvider        = (*podcastPlugin)(nil)
	_ podcast.GetChannelsProvider       = (*podcastPlugin)(nil)
	_ podcast.GetChannelProvider        = (*podcastPlugin)(nil)
	_ podcast.GetNewestEpisodesProvider = (*podcastPlugin)(nil)
	_ podcast.GetEpisodeProvider        = (*podcastPlugin)(nil)
	_ podcast.CreateChannelProvider     = (*podcastPlugin)(nil)
	_ podcast.RefreshChannelsProvider   = (*podcastPlugin)(nil)
	_ podcast.DownloadEpisodeProvider   = (*podcastPlugin)(nil)
	_ podcast.DeleteChannelProvider     = (*podcastPlugin)(nil)
	_ podcast.DeleteEpisodeProvider     = (*podcastPlugin)(nil)
)

// ---------------- Persistence ----------------

// storedChannel is the persisted representation of a channel and its episodes.
type storedChannel struct {
	Channel  podcast.PodcastChannel   `json:"channel"`
	Episodes []podcast.PodcastEpisode `json:"episodes"`
}

func channelKey(id string) string   { return channelKeyPrefix + id }
func channelURLKey(u string) string { return channelURLPrefix + u }

func episodeID(chID, guid string) string {
	h := sha256.Sum256([]byte(chID + ":" + guid))
	return "ep-" + hex.EncodeToString(h[:])[:16]
}

func channelID(feedURL string) string {
	h := sha256.Sum256([]byte(feedURL))
	return "ch-" + hex.EncodeToString(h[:])[:16]
}

// loadChannel reads a channel and its episodes from the KVStore.
func loadChannel(id string) (*storedChannel, bool, error) {
	raw, exists, err := host.KVStoreGet(channelKey(id))
	if err != nil {
		return nil, false, err
	}
	if !exists {
		return nil, false, nil
	}
	var sc storedChannel
	if err := json.Unmarshal(raw, &sc); err != nil {
		return nil, false, err
	}
	return &sc, true, nil
}

func saveChannel(sc *storedChannel) error {
	data, err := json.Marshal(sc)
	if err != nil {
		return err
	}
	if err := host.KVStoreSet(channelKey(sc.Channel.ID), data); err != nil {
		return err
	}
	if err := host.KVStoreSet(channelURLKey(sc.Channel.URL), []byte(sc.Channel.ID)); err != nil {
		return err
	}
	return addToIndex(sc.Channel.ID)
}

func deleteChannelFromStore(id string) error {
	sc, exists, err := loadChannel(id)
	if err != nil {
		return err
	}
	if exists {
		_ = host.KVStoreDelete(channelURLKey(sc.Channel.URL))
	}
	_ = host.KVStoreDelete(channelKey(id))
	return removeFromIndex(id)
}

func listChannelIDs() ([]string, error) {
	raw, exists, err := host.KVStoreGet(indexChannels)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func addToIndex(id string) error {
	ids, err := listChannelIDs()
	if err != nil {
		return err
	}
	for _, existing := range ids {
		if existing == id {
			return nil
		}
	}
	ids = append(ids, id)
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	return host.KVStoreSet(indexChannels, data)
}

func removeFromIndex(id string) error {
	ids, err := listChannelIDs()
	if err != nil {
		return err
	}
	out := ids[:0]
	for _, existing := range ids {
		if existing != id {
			out = append(out, existing)
		}
	}
	data, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return host.KVStoreSet(indexChannels, data)
}

func allStoredChannels(includeEpisodes bool) ([]podcast.PodcastChannel, error) {
	ids, err := listChannelIDs()
	if err != nil {
		return nil, err
	}
	channels := make([]podcast.PodcastChannel, 0, len(ids))
	for _, id := range ids {
		sc, exists, err := loadChannel(id)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		ch := sc.Channel
		if includeEpisodes {
			ch.Episodes = append([]podcast.PodcastEpisode(nil), sc.Episodes...)
		}
		channels = append(channels, ch)
	}
	return channels, nil
}

// ---------------- RSS parsing ----------------

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Image       rssImage  `xml:"image"`
	ItunesImage string    `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	Items       []rssItem `xml:"item"`
}

type rssImage struct {
	URL  string `xml:"url"`
	Href string `xml:"href,attr"`
}

type rssItem struct {
	Title       string       `xml:"title"`
	Description string       `xml:"description"`
	PubDate     string       `xml:"pubDate"`
	GUID        string       `xml:"guid"`
	Enclosure   rssEnclosure `xml:"enclosure"`
	ItunesDur   string       `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd duration"`
}

type rssEnclosure struct {
	URL    string `xml:"url,attr"`
	Length string `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

func fetchFeed(feedURL string) (*rssFeed, error) {
	resp, err := host.HTTPSend(host.HTTPRequest{
		Method:    "GET",
		URL:       feedURL,
		Headers:   map[string]string{"User-Agent": defaultUserAgent, "Accept": "application/rss+xml, application/xml, text/xml, */*"},
		TimeoutMs: 15000,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch feed: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("feed returned HTTP %d", resp.StatusCode)
	}
	var feed rssFeed
	if err := xml.Unmarshal(resp.Body, &feed); err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}
	return &feed, nil
}

// parseRSSDate parses an RFC822/RFC1123 pubDate and returns an ISO 8601 string.
// Falls back to the raw value when parsing fails so clients still see something.
func parseRSSDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		time.RFC822Z,
		time.RFC822,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"02 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC().Format("2006-01-02T15:04:05Z")
		}
	}
	return s
}

func yearFromISO(iso string) int32 {
	if len(iso) >= 4 {
		if y, err := strconv.Atoi(iso[:4]); err == nil {
			return int32(y)
		}
	}
	return 0
}

// imageURLFromFeed picks the best cover image URL from the parsed feed.
func imageURLFromFeed(ch *rssChannel) string {
	if ch.ItunesImage != "" {
		return ch.ItunesImage
	}
	if ch.Image.Href != "" {
		return ch.Image.Href
	}
	if ch.Image.URL != "" {
		return ch.Image.URL
	}
	return ""
}

// enclosureSuffix infers a file suffix from a content type or URL.
func enclosureSuffix(contentType, mediaURL string) string {
	if contentType != "" {
		switch strings.ToLower(contentType) {
		case "audio/mpeg", "audio/mp3":
			return "mp3"
		case "audio/mp4", "audio/m4a", "audio/x-m4a":
			return "m4a"
		case "audio/ogg":
			return "ogg"
		case "audio/opus":
			return "opus"
		case "audio/aac":
			return "aac"
		case "audio/x-wav", "audio/wav":
			return "wav"
		case "audio/flac":
			return "flac"
		}
	}
	if u, err := url.Parse(mediaURL); err == nil {
		p := strings.TrimPrefix(u.Path, "/")
		if dot := strings.LastIndex(p, "."); dot >= 0 {
			suf := strings.ToLower(p[dot+1:])
			if len(suf) <= 5 && isAlphaNum(suf) {
				return suf
			}
		}
		q := u.Query().Get("download")
		if dot := strings.LastIndex(q, "."); dot >= 0 {
			suf := strings.ToLower(q[dot+1:])
			if len(suf) <= 5 && isAlphaNum(suf) {
				return suf
			}
		}
	}
	return "mp3"
}

func isAlphaNum(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}
	return true
}

func atoi32(s string) int32 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return int32(n)
}

func atoi64(s string) int64 {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// itunesDurationToSeconds converts an iTunes duration (seconds or H:MM:SS) to seconds.
func itunesDurationToSeconds(s string) int32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if !strings.Contains(s, ":") {
		return atoi32(s)
	}
	parts := strings.Split(s, ":")
	var total int32
	switch len(parts) {
	case 2:
		total = atoi32(parts[0])*60 + atoi32(parts[1])
	case 3:
		total = atoi32(parts[0])*3600 + atoi32(parts[1])*60 + atoi32(parts[2])
	}
	return total
}

// defaultEpisodeStatus returns the status for a newly fetched episode. Episodes
// whose enclosure URL is reachable for streaming are "completed": they are
// immediately playable from the publisher's URL, so Subsonic clients that only
// surface "completed" episodes (e.g. Tempus) display and stream them. Episodes
// without a usable enclosure fall back to "new".
func defaultEpisodeStatus(enclosureURL string) podcast.PodcastStatus {
	if strings.TrimSpace(enclosureURL) != "" {
		return podcast.PodcastStatusCompleted
	}
	return podcast.PodcastStatusNew
}

// preservedStatus keeps a previously recorded user/system status across feed
// refreshes for states that should not be reset to the default. "completed" is
// also preserved so that an episode verified reachable stays available even if
// the enclosure URL temporarily changes. "new" and "skipped" are NOT preserved:
// a refreshed episode is re-evaluated against its enclosure. Not preserving
// "skipped" is important because it was the legacy default status for
// non-downloaded episodes, so existing data must migrate to the new
// enclosure-based default (typically "completed") rather than stay stuck on
// "skipped", which would hide episodes from Subsonic clients that only surface
// "completed" episodes (e.g. Tempus).
func preservedStatus(prev podcast.PodcastStatus) (podcast.PodcastStatus, bool) {
	switch prev {
	case podcast.PodcastStatusCompleted,
		podcast.PodcastStatusError,
		podcast.PodcastStatusDeleted:
		return prev, true
	}
	return "", false
}

// feedToChannel builds a PodcastChannel from a parsed RSS feed, preserving any
// existing episode download status keyed by episode ID.
func feedToChannel(feedURL string, feed *rssFeed, existing map[string]podcast.PodcastEpisode) podcast.PodcastChannel {
	id := channelID(feedURL)
	img := imageURLFromFeed(&feed.Channel)
	cover := id
	if img == "" {
		cover = ""
	}
	episodes := make([]podcast.PodcastEpisode, 0, len(feed.Channel.Items))
	for _, it := range feed.Channel.Items {
		guid := it.GUID
		if guid == "" {
			guid = it.Enclosure.URL
		}
		if guid == "" && it.Title != "" {
			guid = it.Title
		}
		epID := episodeID(id, guid)
		iso := parseRSSDate(it.PubDate)
		ep := podcast.PodcastEpisode{
			ID:          epID,
			StreamID:    epID,
			ChannelID:   id,
			Title:       it.Title,
			Description: it.Description,
			PublishDate: iso,
			Status:      defaultEpisodeStatus(it.Enclosure.URL),
			StreamURL:   it.Enclosure.URL,
			CoverArt:    cover,
			Year:        yearFromISO(iso),
			Genre:       "Podcast",
			Duration:    itunesDurationToSeconds(it.ItunesDur),
			Size:        atoi64(it.Enclosure.Length),
			ContentType: it.Enclosure.Type,
			Suffix:      enclosureSuffix(it.Enclosure.Type, it.Enclosure.URL),
		}
		if prev, ok := existing[epID]; ok {
			if status, keep := preservedStatus(prev.Status); keep {
				ep.Status = status
			}
			ep.ErrorMessage = prev.ErrorMessage
		}
		episodes = append(episodes, ep)
	}
	sort.SliceStable(episodes, func(i, j int) bool {
		return episodes[i].PublishDate > episodes[j].PublishDate
	})
	return podcast.PodcastChannel{
		ID:               id,
		URL:              feedURL,
		Title:            feed.Channel.Title,
		Description:      feed.Channel.Description,
		CoverArt:         cover,
		OriginalImageUrl: img,
		Status:           podcast.PodcastStatusCompleted,
		Episodes:         episodes,
	}
}

// existingEpisodeMap returns the current download status of episodes for a channel.
func existingEpisodeMap(id string) map[string]podcast.PodcastEpisode {
	sc, exists, err := loadChannel(id)
	if err != nil || !exists {
		return nil
	}
	m := make(map[string]podcast.PodcastEpisode, len(sc.Episodes))
	for _, ep := range sc.Episodes {
		m[ep.ID] = ep
	}
	return m
}

// ---------------- Lifecycle ----------------

func refreshCron() string {
	c, ok := pdk.GetConfig("refreshSchedule")
	if !ok || c == "" {
		return "0 * * * *"
	}
	return c
}

func (p *podcastPlugin) OnInit() error {
	pdk.Log(pdk.LogInfo, "Podcast plugin initializing")
	_, err := host.SchedulerScheduleRecurring(refreshCron(), "refresh-all", scheduleID)
	if err != nil {
		pdk.Log(pdk.LogWarn, fmt.Sprintf("failed to schedule refresh: %v", err))
	}
	return nil
}

// ---------------- Scheduler callback ----------------

func (p *podcastPlugin) OnCallback(input scheduler.SchedulerCallbackRequest) error {
	if input.ScheduleID != scheduleID && input.Payload != "refresh-all" {
		return nil
	}
	pdk.Log(pdk.LogDebug, "Refreshing all podcast feeds")
	ids, err := listChannelIDs()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := refreshChannelByID(id); err != nil {
			pdk.Log(pdk.LogWarn, fmt.Sprintf("refresh %s failed: %v", id, err))
		}
	}
	return nil
}

func refreshChannelByID(id string) (string, error) {
	sc, exists, err := loadChannel(id)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", errors.New("channel not found")
	}
	feed, err := fetchFeed(sc.Channel.URL)
	if err != nil {
		sc.Channel.Status = podcast.PodcastStatusError
		sc.Channel.ErrorMessage = err.Error()
		_ = saveChannel(sc)
		return id, err
	}
	updated := feedToChannel(sc.Channel.URL, feed, existingEpisodeMap(id))
	sc.Channel = updated
	sc.Channel.Status = podcast.PodcastStatusCompleted
	sc.Channel.ErrorMessage = ""
	sc.Episodes = updated.Episodes
	if err := saveChannel(sc); err != nil {
		return "", err
	}
	return id, nil
}

// ---------------- Podcast capability ----------------

func (p *podcastPlugin) GetChannels(req podcast.GetPodcastChannelsRequest) (*podcast.GetPodcastChannelsResponse, error) {
	channels, err := allStoredChannels(req.IncludeEpisodes)
	if err != nil {
		return nil, err
	}
	return &podcast.GetPodcastChannelsResponse{Channels: channels}, nil
}

func (p *podcastPlugin) GetChannel(req podcast.GetPodcastChannelRequest) (*podcast.GetPodcastChannelResponse, error) {
	sc, exists, err := loadChannel(req.ID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return &podcast.GetPodcastChannelResponse{}, nil
	}
	ch := sc.Channel
	if req.IncludeEpisodes {
		ch.Episodes = append([]podcast.PodcastEpisode(nil), sc.Episodes...)
	}
	return &podcast.GetPodcastChannelResponse{Channel: ch}, nil
}

func (p *podcastPlugin) GetNewestEpisodes(req podcast.GetNewestEpisodesRequest) (*podcast.GetNewestEpisodesResponse, error) {
	ids, err := listChannelIDs()
	if err != nil {
		return nil, err
	}
	var all []podcast.PodcastEpisode
	for _, id := range ids {
		sc, exists, err := loadChannel(id)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		all = append(all, sc.Episodes...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].PublishDate > all[j].PublishDate
	})
	if req.Count > 0 && int(req.Count) < len(all) {
		all = all[:req.Count]
	}
	return &podcast.GetNewestEpisodesResponse{Episodes: all}, nil
}

func (p *podcastPlugin) GetEpisode(req podcast.GetPodcastEpisodeRequest) (*podcast.GetPodcastEpisodeResponse, error) {
	ids, err := listChannelIDs()
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		sc, exists, err := loadChannel(id)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		for _, ep := range sc.Episodes {
			if ep.ID == req.ID {
				return &podcast.GetPodcastEpisodeResponse{Episode: &ep}, nil
			}
		}
	}
	return &podcast.GetPodcastEpisodeResponse{}, nil
}

func (p *podcastPlugin) CreateChannel(req podcast.CreatePodcastChannelRequest) (*podcast.CreatePodcastChannelResponse, error) {
	if strings.TrimSpace(req.URL) == "" {
		return nil, errors.New("feed URL is required")
	}
	existing, exists, err := host.KVStoreGet(channelURLKey(req.URL))
	if err != nil {
		return nil, err
	}
	if exists {
		sc, ok, err := loadChannel(string(existing))
		if err != nil {
			return nil, err
		}
		if ok {
			return &podcast.CreatePodcastChannelResponse{Channel: &sc.Channel}, nil
		}
	}
	feed, err := fetchFeed(req.URL)
	if err != nil {
		return nil, err
	}
	ch := feedToChannel(req.URL, feed, nil)
	sc := &storedChannel{Channel: ch, Episodes: ch.Episodes}
	if err := saveChannel(sc); err != nil {
		return nil, err
	}
	pdk.Log(pdk.LogInfo, fmt.Sprintf("Subscribed to podcast: %s (%s)", ch.Title, req.URL))
	return &podcast.CreatePodcastChannelResponse{Channel: &ch}, nil
}

func (p *podcastPlugin) RefreshChannels(req podcast.RefreshPodcastsRequest) (*podcast.RefreshPodcastsResponse, error) {
	ids := req.ChannelIDs
	if len(ids) == 0 {
		var err error
		ids, err = listChannelIDs()
		if err != nil {
			return nil, err
		}
	}
	var refreshed []string
	for _, id := range ids {
		if rid, err := refreshChannelByID(id); err == nil {
			refreshed = append(refreshed, rid)
		} else {
			pdk.Log(pdk.LogWarn, fmt.Sprintf("refresh %s failed: %v", id, err))
		}
	}
	return &podcast.RefreshPodcastsResponse{Refreshed: refreshed}, nil
}

func (p *podcastPlugin) DownloadEpisode(req podcast.DownloadPodcastEpisodeRequest) (*podcast.DownloadPodcastEpisodeResponse, error) {
	ids, err := listChannelIDs()
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		sc, exists, err := loadChannel(id)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		for i := range sc.Episodes {
			if sc.Episodes[i].ID == req.ID {
				sc.Episodes[i].Status = podcast.PodcastStatusDownloading
				sc.Episodes[i].ErrorMessage = ""
				if err := saveChannel(sc); err != nil {
					return nil, err
				}
				resp, err := host.HTTPSend(host.HTTPRequest{
					Method:    "HEAD",
					URL:       sc.Episodes[i].StreamURL,
					Headers:   map[string]string{"User-Agent": defaultUserAgent},
					TimeoutMs: 10000,
				})
				if err != nil {
					sc.Episodes[i].Status = podcast.PodcastStatusError
					sc.Episodes[i].ErrorMessage = err.Error()
				} else if resp.StatusCode >= 200 && resp.StatusCode < 400 {
					sc.Episodes[i].Status = podcast.PodcastStatusCompleted
					sc.Episodes[i].ErrorMessage = ""
				} else {
					sc.Episodes[i].Status = podcast.PodcastStatusError
					sc.Episodes[i].ErrorMessage = fmt.Sprintf("enclosure returned HTTP %d", resp.StatusCode)
				}
				if err := saveChannel(sc); err != nil {
					return nil, err
				}
				ep := sc.Episodes[i]
				return &podcast.DownloadPodcastEpisodeResponse{Episode: &ep}, nil
			}
		}
	}
	return &podcast.DownloadPodcastEpisodeResponse{}, nil
}

func (p *podcastPlugin) DeleteChannel(req podcast.DeletePodcastChannelRequest) (*podcast.DeletePodcastChannelResponse, error) {
	if err := deleteChannelFromStore(req.ID); err != nil {
		return &podcast.DeletePodcastChannelResponse{Deleted: false}, err
	}
	return &podcast.DeletePodcastChannelResponse{Deleted: true}, nil
}

func (p *podcastPlugin) DeleteEpisode(req podcast.DeletePodcastEpisodeRequest) (*podcast.DeletePodcastEpisodeResponse, error) {
	ids, err := listChannelIDs()
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		sc, exists, err := loadChannel(id)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		for i := range sc.Episodes {
			if sc.Episodes[i].ID == req.ID {
				sc.Episodes = append(sc.Episodes[:i], sc.Episodes[i+1:]...)
				if err := saveChannel(sc); err != nil {
					return nil, err
				}
				return &podcast.DeletePodcastEpisodeResponse{Deleted: true}, nil
			}
		}
	}
	return &podcast.DeletePodcastEpisodeResponse{Deleted: false}, nil
}

func main() {}
