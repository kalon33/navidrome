package podcast

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

const capabilityPodcast = "Podcast"

// ErrNoProvider is returned when no podcast plugin is configured.
var ErrNoProvider = errors.New("no podcast provider plugin configured")

// Provider is the podcast backend surface implemented by plugin adapters.
type Provider interface {
	GetChannels(ctx context.Context, includeEpisodes bool) ([]capabilities.PodcastChannel, error)
	GetChannel(ctx context.Context, id string, includeEpisodes bool) (*capabilities.PodcastChannel, error)
	GetEpisode(ctx context.Context, id string) (*capabilities.PodcastEpisode, error)
	GetNewestEpisodes(ctx context.Context, count int) ([]capabilities.PodcastEpisode, error)
	CreateChannel(ctx context.Context, url string) (*capabilities.PodcastChannel, error)
	RefreshChannels(ctx context.Context, channelIDs []string) ([]string, error)
	DownloadEpisode(ctx context.Context, id string) (*capabilities.PodcastEpisode, error)
	DeleteChannel(ctx context.Context, id string) error
	DeleteEpisode(ctx context.Context, id string) error
}

// PluginLoader discovers and loads podcast provider plugins.
type PluginLoader interface {
	PluginNames(capability string) []string
	LoadPodcast(name string) (Provider, bool)
}

// Podcast is the podcast service facade the API layers depend on.
type Podcast struct {
	pluginLoader PluginLoader
}

// New creates a Podcast service backed by the given plugin loader.
func New(pluginLoader PluginLoader) *Podcast {
	return &Podcast{pluginLoader: pluginLoader}
}

// Engine is the podcast surface the API layers depend on; *Podcast satisfies it.
type Engine interface {
	HasProvider() bool
	GetChannels(ctx context.Context, includeEpisodes bool) ([]capabilities.PodcastChannel, error)
	GetChannel(ctx context.Context, id string, includeEpisodes bool) (*capabilities.PodcastChannel, error)
	GetEpisode(ctx context.Context, id string) (*capabilities.PodcastEpisode, error)
	GetNewestEpisodes(ctx context.Context, count int) ([]capabilities.PodcastEpisode, error)
	CreateChannel(ctx context.Context, url string) (*capabilities.PodcastChannel, error)
	RefreshChannels(ctx context.Context, channelIDs []string) ([]string, error)
	DownloadEpisode(ctx context.Context, id string) (*capabilities.PodcastEpisode, error)
	DeleteChannel(ctx context.Context, id string) error
	DeleteEpisode(ctx context.Context, id string) error
}

var _ Engine = (*Podcast)(nil)

func (p *Podcast) HasProvider() bool {
	return len(p.pluginLoader.PluginNames(capabilityPodcast)) > 0
}

func (p *Podcast) loadProvider() (Provider, error) {
	names := p.pluginLoader.PluginNames(capabilityPodcast)
	if len(names) == 0 {
		return nil, ErrNoProvider
	}
	provider, ok := p.pluginLoader.LoadPodcast(names[0])
	if !ok {
		return nil, ErrNoProvider
	}
	return provider, nil
}

func (p *Podcast) GetChannels(ctx context.Context, includeEpisodes bool) ([]capabilities.PodcastChannel, error) {
	provider, err := p.loadProvider()
	if err != nil {
		return nil, err
	}
	channels, err := provider.GetChannels(ctx, includeEpisodes)
	if err != nil {
		log.Error(ctx, "Plugin GetChannels failed", err)
		return nil, err
	}
	return channels, nil
}

func (p *Podcast) GetChannel(ctx context.Context, id string, includeEpisodes bool) (*capabilities.PodcastChannel, error) {
	provider, err := p.loadProvider()
	if err != nil {
		return nil, err
	}
	channel, err := provider.GetChannel(ctx, id, includeEpisodes)
	if err != nil {
		log.Error(ctx, "Plugin GetChannel failed", "id", id, err)
		return nil, err
	}
	return channel, nil
}

func (p *Podcast) GetEpisode(ctx context.Context, id string) (*capabilities.PodcastEpisode, error) {
	provider, err := p.loadProvider()
	if err != nil {
		return nil, err
	}
	episode, err := provider.GetEpisode(ctx, id)
	if err != nil {
		log.Error(ctx, "Plugin GetEpisode failed", "id", id, err)
		return nil, err
	}
	return episode, nil
}

func (p *Podcast) GetNewestEpisodes(ctx context.Context, count int) ([]capabilities.PodcastEpisode, error) {
	provider, err := p.loadProvider()
	if err != nil {
		return nil, err
	}
	episodes, err := provider.GetNewestEpisodes(ctx, count)
	if err != nil {
		log.Error(ctx, "Plugin GetNewestEpisodes failed", err)
		return nil, err
	}
	return episodes, nil
}

func (p *Podcast) CreateChannel(ctx context.Context, url string) (*capabilities.PodcastChannel, error) {
	provider, err := p.loadProvider()
	if err != nil {
		return nil, err
	}
	channel, err := provider.CreateChannel(ctx, url)
	if err != nil {
		log.Error(ctx, "Plugin CreateChannel failed", "url", url, err)
		return nil, err
	}
	return channel, nil
}

func (p *Podcast) RefreshChannels(ctx context.Context, channelIDs []string) ([]string, error) {
	provider, err := p.loadProvider()
	if err != nil {
		return nil, err
	}
	refreshed, err := provider.RefreshChannels(ctx, channelIDs)
	if err != nil {
		log.Error(ctx, "Plugin RefreshChannels failed", err)
		return nil, err
	}
	return refreshed, nil
}

func (p *Podcast) DownloadEpisode(ctx context.Context, id string) (*capabilities.PodcastEpisode, error) {
	provider, err := p.loadProvider()
	if err != nil {
		return nil, err
	}
	episode, err := provider.DownloadEpisode(ctx, id)
	if err != nil {
		log.Error(ctx, "Plugin DownloadEpisode failed", "id", id, err)
		return nil, err
	}
	return episode, nil
}

func (p *Podcast) DeleteChannel(ctx context.Context, id string) error {
	provider, err := p.loadProvider()
	if err != nil {
		return err
	}
	if err := provider.DeleteChannel(ctx, id); err != nil {
		log.Error(ctx, "Plugin DeleteChannel failed", "id", id, err)
		return err
	}
	return nil
}

func (p *Podcast) DeleteEpisode(ctx context.Context, id string) error {
	provider, err := p.loadProvider()
	if err != nil {
		return err
	}
	if err := provider.DeleteEpisode(ctx, id); err != nil {
		log.Error(ctx, "Plugin DeleteEpisode failed", "id", id, err)
		return err
	}
	return nil
}
