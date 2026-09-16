package subsonic

import (
	"net/http"

	"github.com/navidrome/navidrome/core/podcast"
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

	resp := make([]responses.PodcastEpisode, len(episodes))
	for i, ep := range episodes {
		resp[i] = toPodcastEpisode(ep)
	}
	response := newResponse()
	response.NewestPodcasts = &responses.NewestPodcasts{Episodes: resp}
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
		CoverArt:         ch.CoverArt,
		OriginalImageUrl: ch.OriginalImageUrl,
		Status:           string(ch.Status),
		ErrorMessage:     ch.ErrorMessage,
	}
	if len(ch.Episodes) > 0 {
		resp.Episode = make([]responses.PodcastEpisode, len(ch.Episodes))
		for i, ep := range ch.Episodes {
			resp.Episode[i] = toPodcastEpisode(ep)
		}
	}
	return resp
}

func toPodcastEpisode(ep capabilities.PodcastEpisode) responses.PodcastEpisode {
	resp := responses.PodcastEpisode{
		Child: responses.Child{
			Id:          ep.ID,
			Title:       ep.Title,
			IsDir:       false,
			Year:        ep.Year,
			Genre:       ep.Genre,
			CoverArt:    ep.CoverArt,
			Size:        ep.Size,
			ContentType: ep.ContentType,
			Suffix:      ep.Suffix,
			Duration:    ep.Duration,
			BitRate:     ep.BitRate,
			Path:        ep.Path,
			Type:        "podcast",
		},
		StreamId:    ep.StreamID,
		ChannelId:   ep.ChannelID,
		Description: ep.Description,
		Status:      string(ep.Status),
		PublishDate: ep.PublishDate,
	}
	return resp
}
