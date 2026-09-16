package podcast_test

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/core/podcast"
	"github.com/navidrome/navidrome/plugins/capabilities"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockPluginLoader struct {
	names    []string
	provider podcast.Provider
	loadOk   bool
}

func (m *mockPluginLoader) PluginNames(capability string) []string {
	if capability == "Podcast" {
		return m.names
	}
	return nil
}

func (m *mockPluginLoader) LoadPodcast(_ string) (podcast.Provider, bool) {
	return m.provider, m.loadOk
}

type mockProvider struct {
	channels       []capabilities.PodcastChannel
	channel        *capabilities.PodcastChannel
	newestEpisodes []capabilities.PodcastEpisode
	episode        *capabilities.PodcastEpisode
	createErr      error
	refreshedIDs   []string
	downloadEp     *capabilities.PodcastEpisode
	deleteErr      error
	lastChannelID  string
	lastEpisodeID  string
}

func (m *mockProvider) GetChannels(_ context.Context, _ bool) ([]capabilities.PodcastChannel, error) {
	return m.channels, nil
}
func (m *mockProvider) GetChannel(_ context.Context, _ string, _ bool) (*capabilities.PodcastChannel, error) {
	return m.channel, nil
}
func (m *mockProvider) GetNewestEpisodes(_ context.Context, _ int) ([]capabilities.PodcastEpisode, error) {
	return m.newestEpisodes, nil
}
func (m *mockProvider) GetEpisode(_ context.Context, _ string) (*capabilities.PodcastEpisode, error) {
	return m.episode, nil
}
func (m *mockProvider) CreateChannel(_ context.Context, _ string) (*capabilities.PodcastChannel, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return &capabilities.PodcastChannel{ID: "new"}, nil
}
func (m *mockProvider) RefreshChannels(_ context.Context, _ []string) ([]string, error) {
	return m.refreshedIDs, nil
}
func (m *mockProvider) DownloadEpisode(_ context.Context, _ string) (*capabilities.PodcastEpisode, error) {
	return m.downloadEp, nil
}
func (m *mockProvider) DeleteChannel(_ context.Context, id string) error {
	m.lastChannelID = id
	return m.deleteErr
}
func (m *mockProvider) DeleteEpisode(_ context.Context, id string) error {
	m.lastEpisodeID = id
	return m.deleteErr
}

var _ = Describe("Podcast", func() {
	var (
		ctx     context.Context
		loader  *mockPluginLoader
		provider *mockProvider
		service *podcast.Podcast
	)

	BeforeEach(func() {
		ctx = context.Background()
		provider = &mockProvider{}
		loader = &mockPluginLoader{provider: provider, loadOk: true}
	})

	Describe("without a provider", func() {
		BeforeEach(func() {
			loader.names = nil
			service = podcast.New(loader)
		})
		It("HasProvider returns false", func() {
			Expect(service.HasProvider()).To(BeFalse())
		})
		It("GetChannels returns ErrNoProvider", func() {
			_, err := service.GetChannels(ctx, true)
			Expect(err).To(MatchError(podcast.ErrNoProvider))
		})
	})

	Describe("with a provider", func() {
		BeforeEach(func() {
			loader.names = []string{"test-plugin"}
			service = podcast.New(loader)
		})
		It("HasProvider returns true", func() {
			Expect(service.HasProvider()).To(BeTrue())
		})
		It("GetChannels returns channels", func() {
			provider.channels = []capabilities.PodcastChannel{{ID: "ch-1", Status: capabilities.PodcastStatusCompleted}}
			channels, err := service.GetChannels(ctx, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(channels).To(HaveLen(1))
			Expect(channels[0].ID).To(Equal("ch-1"))
		})
		It("CreateChannel returns the created channel", func() {
			ch, err := service.CreateChannel(ctx, "https://example.com/feed.xml")
			Expect(err).ToNot(HaveOccurred())
			Expect(ch.ID).To(Equal("new"))
		})
		It("CreateChannel propagates errors", func() {
			provider.createErr = errors.New("boom")
			_, err := service.CreateChannel(ctx, "https://example.com/feed.xml")
			Expect(err).To(HaveOccurred())
		})
		It("DeleteChannel delegates to the provider", func() {
			Expect(service.DeleteChannel(ctx, "ch-1")).To(Succeed())
			Expect(provider.lastChannelID).To(Equal("ch-1"))
		})
		It("DeleteEpisode delegates to the provider", func() {
			Expect(service.DeleteEpisode(ctx, "ep-1")).To(Succeed())
			Expect(provider.lastEpisodeID).To(Equal("ep-1"))
		})
		It("GetEpisode delegates to the provider", func() {
			provider.episode = &capabilities.PodcastEpisode{ID: "ep-1"}
			ep, err := service.GetEpisode(ctx, "ep-1")
			Expect(err).ToNot(HaveOccurred())
			Expect(ep.ID).To(Equal("ep-1"))
		})
	})
})
