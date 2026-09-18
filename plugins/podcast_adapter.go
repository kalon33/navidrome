package plugins

import (
	"context"

	"github.com/navidrome/navidrome/core/podcast"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

const CapabilityPodcast Capability = "Podcast"

const (
	FuncPodcastGetChannels       = "nd_podcast_get_channels"
	FuncPodcastGetChannel        = "nd_podcast_get_channel"
	FuncPodcastGetEpisode        = "nd_podcast_get_episode"
	FuncPodcastGetNewestEpisodes = "nd_podcast_get_newest_episodes"
	FuncPodcastCreateChannel     = "nd_podcast_create_channel"
	FuncPodcastRefreshChannels   = "nd_podcast_refresh_channels"
	FuncPodcastDownloadEpisode   = "nd_podcast_download_episode"
	FuncPodcastDeleteChannel     = "nd_podcast_delete_channel"
	FuncPodcastDeleteEpisode     = "nd_podcast_delete_episode"
)

func init() {
	registerCapability(
		CapabilityPodcast,
		FuncPodcastGetChannels,
		FuncPodcastGetChannel,
		FuncPodcastGetEpisode,
		FuncPodcastGetNewestEpisodes,
		FuncPodcastCreateChannel,
		FuncPodcastRefreshChannels,
		FuncPodcastDownloadEpisode,
		FuncPodcastDeleteChannel,
		FuncPodcastDeleteEpisode,
	)
}

func newPodcastPlugin(p *plugin) *PodcastPlugin {
	return &PodcastPlugin{name: p.name, plugin: p}
}

// PodcastPlugin adapts a WASM plugin with the Podcast capability.
type PodcastPlugin struct {
	name   string
	plugin *plugin
}

func (p *PodcastPlugin) GetChannels(ctx context.Context, includeEpisodes bool) ([]capabilities.PodcastChannel, error) {
	req := capabilities.GetPodcastChannelsRequest{IncludeEpisodes: includeEpisodes}
	resp, err := callPluginFunction[capabilities.GetPodcastChannelsRequest, capabilities.GetPodcastChannelsResponse](
		ctx, p.plugin, FuncPodcastGetChannels, req,
	)
	if err != nil {
		return nil, err
	}
	return resp.Channels, nil
}

func (p *PodcastPlugin) GetChannel(ctx context.Context, id string, includeEpisodes bool) (*capabilities.PodcastChannel, error) {
	req := capabilities.GetPodcastChannelRequest{ID: id, IncludeEpisodes: includeEpisodes}
	resp, err := callPluginFunction[capabilities.GetPodcastChannelRequest, capabilities.GetPodcastChannelResponse](
		ctx, p.plugin, FuncPodcastGetChannel, req,
	)
	if err != nil {
		return nil, err
	}
	return &resp.Channel, nil
}

func (p *PodcastPlugin) GetEpisode(ctx context.Context, id string) (*capabilities.PodcastEpisode, error) {
	req := capabilities.GetPodcastEpisodeRequest{ID: id}
	resp, err := callPluginFunction[capabilities.GetPodcastEpisodeRequest, capabilities.GetPodcastEpisodeResponse](
		ctx, p.plugin, FuncPodcastGetEpisode, req,
	)
	if err != nil {
		return nil, err
	}
	return resp.Episode, nil
}

func (p *PodcastPlugin) GetNewestEpisodes(ctx context.Context, count int) ([]capabilities.PodcastEpisode, error) {
	req := capabilities.GetNewestEpisodesRequest{Count: int32(count)}
	resp, err := callPluginFunction[capabilities.GetNewestEpisodesRequest, capabilities.GetNewestEpisodesResponse](
		ctx, p.plugin, FuncPodcastGetNewestEpisodes, req,
	)
	if err != nil {
		return nil, err
	}
	return resp.Episodes, nil
}

func (p *PodcastPlugin) CreateChannel(ctx context.Context, url string) (*capabilities.PodcastChannel, error) {
	req := capabilities.CreatePodcastChannelRequest{URL: url}
	resp, err := callPluginFunction[capabilities.CreatePodcastChannelRequest, capabilities.CreatePodcastChannelResponse](
		ctx, p.plugin, FuncPodcastCreateChannel, req,
	)
	if err != nil {
		return nil, err
	}
	return resp.Channel, nil
}

func (p *PodcastPlugin) RefreshChannels(ctx context.Context, channelIDs []string) ([]string, error) {
	req := capabilities.RefreshPodcastsRequest{ChannelIDs: channelIDs}
	resp, err := callPluginFunction[capabilities.RefreshPodcastsRequest, capabilities.RefreshPodcastsResponse](
		ctx, p.plugin, FuncPodcastRefreshChannels, req,
	)
	if err != nil {
		return nil, err
	}
	return resp.Refreshed, nil
}

func (p *PodcastPlugin) DownloadEpisode(ctx context.Context, id string) (*capabilities.PodcastEpisode, error) {
	req := capabilities.DownloadPodcastEpisodeRequest{ID: id}
	resp, err := callPluginFunction[capabilities.DownloadPodcastEpisodeRequest, capabilities.DownloadPodcastEpisodeResponse](
		ctx, p.plugin, FuncPodcastDownloadEpisode, req,
	)
	if err != nil {
		return nil, err
	}
	return resp.Episode, nil
}

func (p *PodcastPlugin) DeleteChannel(ctx context.Context, id string) error {
	req := capabilities.DeletePodcastChannelRequest{ID: id}
	_, err := callPluginFunction[capabilities.DeletePodcastChannelRequest, capabilities.DeletePodcastChannelResponse](
		ctx, p.plugin, FuncPodcastDeleteChannel, req,
	)
	return err
}

func (p *PodcastPlugin) DeleteEpisode(ctx context.Context, id string) error {
	req := capabilities.DeletePodcastEpisodeRequest{ID: id}
	_, err := callPluginFunction[capabilities.DeletePodcastEpisodeRequest, capabilities.DeletePodcastEpisodeResponse](
		ctx, p.plugin, FuncPodcastDeleteEpisode, req,
	)
	return err
}

var _ podcast.Provider = (*PodcastPlugin)(nil)
