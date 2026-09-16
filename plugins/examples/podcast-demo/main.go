// Demo Navidrome podcast plugin.
//
// This is a minimal, in-memory podcast backend that demonstrates the Podcast
// capability. It serves a single hard-coded channel with one episode, and
// supports the full set of podcast management operations (create channel,
// refresh, download, delete) as no-ops against the in-memory store.
//
// It is intended as a reference implementation: real backends should use the
// HTTP host service to fetch and parse RSS feeds, the KVStore to persist
// channels/episodes, and the Scheduler to refresh feeds periodically.
//
// Build with:
//
//	tinygo build -o podcast-demo.wasm -target wasip1 -buildmode=c-shared .
//	zip -j podcast-demo.ndp manifest.json podcast-demo.wasm
package main

import (
	"github.com/navidrome/navidrome/plugins/pdk/go/podcast"
)

type demoPlugin struct{}

func init() {
	podcast.Register(&demoPlugin{})
}

var (
	_ podcast.GetChannelsProvider       = (*demoPlugin)(nil)
	_ podcast.GetChannelProvider        = (*demoPlugin)(nil)
	_ podcast.GetNewestEpisodesProvider = (*demoPlugin)(nil)
	_ podcast.GetEpisodeProvider        = (*demoPlugin)(nil)
	_ podcast.CreateChannelProvider     = (*demoPlugin)(nil)
	_ podcast.RefreshChannelsProvider   = (*demoPlugin)(nil)
	_ podcast.DownloadEpisodeProvider   = (*demoPlugin)(nil)
	_ podcast.DeleteChannelProvider     = (*demoPlugin)(nil)
	_ podcast.DeleteEpisodeProvider     = (*demoPlugin)(nil)
)

func sampleChannel() podcast.PodcastChannel {
	return podcast.PodcastChannel{
		ID:               "demo-channel-1",
		URL:              "https://example.com/podcast/rss.xml",
		Title:            "Navidrome Demo Podcast",
		Description:      "A demo podcast channel served by the podcast-demo plugin.",
		CoverArt:         "pod-demo-channel-1",
		OriginalImageUrl: "https://example.com/podcast/cover.jpg",
		Status:           podcast.PodcastStatusCompleted,
		Episodes: []podcast.PodcastEpisode{
			{
				ID:          "demo-episode-1",
				StreamID:    "demo-episode-1",
				ChannelID:   "demo-channel-1",
				Title:       "Welcome to the Demo Podcast",
				Description: "The first episode of the Navidrome demo podcast.",
				PublishDate: "2024-01-01T00:00:00Z",
				Status:      podcast.PodcastStatusCompleted,
				StreamURL:   "https://example.com/podcast/episode-1.mp3",
				CoverArt:    "pod-demo-channel-1",
				Year:        2024,
				Genre:       "Podcast",
				Duration:    600,
				BitRate:     128,
				Size:        9600000,
				ContentType: "audio/mpeg",
				Suffix:      "mp3",
			},
		},
	}
}

func (p *demoPlugin) GetChannels(req podcast.GetPodcastChannelsRequest) (*podcast.GetPodcastChannelsResponse, error) {
	ch := sampleChannel()
	if !req.IncludeEpisodes {
		ch.Episodes = nil
	}
	return &podcast.GetPodcastChannelsResponse{Channels: []podcast.PodcastChannel{ch}}, nil
}

func (p *demoPlugin) GetChannel(req podcast.GetPodcastChannelRequest) (*podcast.GetPodcastChannelResponse, error) {
	ch := sampleChannel()
	if !req.IncludeEpisodes {
		ch.Episodes = nil
	}
	return &podcast.GetPodcastChannelResponse{Channel: ch}, nil
}

func (p *demoPlugin) GetNewestEpisodes(req podcast.GetNewestEpisodesRequest) (*podcast.GetNewestEpisodesResponse, error) {
	ch := sampleChannel()
	episodes := ch.Episodes
	if int(req.Count) > 0 && int(req.Count) < len(episodes) {
		episodes = episodes[:req.Count]
	}
	return &podcast.GetNewestEpisodesResponse{Episodes: episodes}, nil
}

func (p *demoPlugin) GetEpisode(req podcast.GetPodcastEpisodeRequest) (*podcast.GetPodcastEpisodeResponse, error) {
	ch := sampleChannel()
	for _, ep := range ch.Episodes {
		if ep.ID == req.ID {
			return &podcast.GetPodcastEpisodeResponse{Episode: &ep}, nil
		}
	}
	return &podcast.GetPodcastEpisodeResponse{}, nil
}

func (p *demoPlugin) CreateChannel(req podcast.CreatePodcastChannelRequest) (*podcast.CreatePodcastChannelResponse, error) {
	ch := podcast.PodcastChannel{
		ID:     "demo-channel-2",
		URL:    req.URL,
		Title:  "New Channel",
		Status: podcast.PodcastStatusNew,
	}
	return &podcast.CreatePodcastChannelResponse{Channel: &ch}, nil
}

func (p *demoPlugin) RefreshChannels(_ podcast.RefreshPodcastsRequest) (*podcast.RefreshPodcastsResponse, error) {
	return &podcast.RefreshPodcastsResponse{Refreshed: []string{"demo-channel-1"}}, nil
}

func (p *demoPlugin) DownloadEpisode(_ podcast.DownloadPodcastEpisodeRequest) (*podcast.DownloadPodcastEpisodeResponse, error) {
	ch := sampleChannel()
	if len(ch.Episodes) > 0 {
		ep := ch.Episodes[0]
		ep.Status = podcast.PodcastStatusDownloading
		return &podcast.DownloadPodcastEpisodeResponse{Episode: &ep}, nil
	}
	return &podcast.DownloadPodcastEpisodeResponse{}, nil
}

func (p *demoPlugin) DeleteChannel(_ podcast.DeletePodcastChannelRequest) (*podcast.DeletePodcastChannelResponse, error) {
	return &podcast.DeletePodcastChannelResponse{Deleted: true}, nil
}

func (p *demoPlugin) DeleteEpisode(_ podcast.DeletePodcastEpisodeRequest) (*podcast.DeletePodcastEpisodeResponse, error) {
	return &podcast.DeletePodcastEpisodeResponse{Deleted: true}, nil
}

func main() {}
