package subsonic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/navidrome/navidrome/plugins/capabilities"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakePodcastEngine is a test double for podcastsvc.Engine that returns canned
// data and records the calls it receives.
type fakePodcastEngine struct {
	hasProvider        bool
	channels           []capabilities.PodcastChannel
	channel            *capabilities.PodcastChannel
	newestEpisodes     []capabilities.PodcastEpisode
	episode            *capabilities.PodcastEpisode
	createErr          error
	createdChannel     *capabilities.PodcastChannel
	refreshedIDs       []string
	downloadedEpisode  *capabilities.PodcastEpisode
	deleteErr          error
	lastChannelID      string
	lastEpisodeID      string
}

func (f *fakePodcastEngine) HasProvider() bool { return f.hasProvider }
func (f *fakePodcastEngine) GetChannels(_ context.Context, _ bool) ([]capabilities.PodcastChannel, error) {
	return f.channels, nil
}
func (f *fakePodcastEngine) GetChannel(_ context.Context, id string, _ bool) (*capabilities.PodcastChannel, error) {
	f.lastChannelID = id
	return f.channel, nil
}
func (f *fakePodcastEngine) GetNewestEpisodes(_ context.Context, _ int) ([]capabilities.PodcastEpisode, error) {
	return f.newestEpisodes, nil
}
func (f *fakePodcastEngine) GetEpisode(_ context.Context, _ string) (*capabilities.PodcastEpisode, error) {
	return f.episode, nil
}
func (f *fakePodcastEngine) CreateChannel(_ context.Context, _ string) (*capabilities.PodcastChannel, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.createdChannel, nil
}
func (f *fakePodcastEngine) RefreshChannels(_ context.Context, _ []string) ([]string, error) {
	return f.refreshedIDs, nil
}
func (f *fakePodcastEngine) DownloadEpisode(_ context.Context, id string) (*capabilities.PodcastEpisode, error) {
	f.lastEpisodeID = id
	return f.downloadedEpisode, nil
}
func (f *fakePodcastEngine) DeleteChannel(_ context.Context, id string) error {
	f.lastChannelID = id
	return f.deleteErr
}
func (f *fakePodcastEngine) DeleteEpisode(_ context.Context, id string) error {
	f.lastEpisodeID = id
	return f.deleteErr
}

func newPodcastRequest(endpoint string, params ...string) *http.Request {
	v := url.Values{}
	for i := 0; i < len(params); i += 2 {
		v.Set(params[i], params[i+1])
	}
	r := httptest.NewRequest("GET", "/rest/"+endpoint+"?"+v.Encode(), nil)
	return r
}

var _ = Describe("Podcasts", func() {
	var api *Router

	Describe("without a provider", func() {
		BeforeEach(func() {
			api = &Router{podcast: &fakePodcastEngine{hasProvider: false}}
		})

		It("getPodcasts returns an error when no provider", func() {
			_, err := api.GetPodcasts(newPodcastRequest("getPodcasts"))
			Expect(err).To(HaveOccurred())
		})

		It("getNewestPodcasts returns an error when no provider", func() {
			_, err := api.GetNewestPodcasts(newPodcastRequest("getNewestPodcasts"))
			Expect(err).To(HaveOccurred())
		})

		It("getPodcastEpisode returns an error when no provider", func() {
			_, err := api.GetPodcastEpisode(newPodcastRequest("getPodcastEpisode", "id", "ep-1"))
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("with a provider", func() {
		var engine *fakePodcastEngine

		BeforeEach(func() {
			engine = &fakePodcastEngine{
				hasProvider: true,
				channels: []capabilities.PodcastChannel{
					{
						ID:     "ch-1",
						URL:    "https://example.com/feed.xml",
						Title:  "Test Channel",
						Status: capabilities.PodcastStatusCompleted,
						Episodes: []capabilities.PodcastEpisode{
							{
								ID:          "ep-1",
								StreamID:    "ep-1",
								ChannelID:   "ch-1",
								Title:       "Episode 1",
								Status:      capabilities.PodcastStatusCompleted,
								PublishDate: "2024-01-02T00:00:00Z",
							},
						},
					},
				},
				channel: &capabilities.PodcastChannel{
					ID:     "ch-1",
					URL:    "https://example.com/feed.xml",
					Title:  "Test Channel",
					Status: capabilities.PodcastStatusCompleted,
				},
				newestEpisodes: []capabilities.PodcastEpisode{
					{
						ID:        "ep-1",
						ChannelID: "ch-1",
						Title:     "Episode 1",
						Status:    capabilities.PodcastStatusCompleted,
					},
				},
			}
			api = &Router{podcast: engine}
		})

		It("getPodcasts returns channels with episodes", func() {
			resp, err := api.GetPodcasts(newPodcastRequest("getPodcasts", "includeEpisodes", "true"))
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Podcasts).ToNot(BeNil())
			Expect(resp.Podcasts.Channels).To(HaveLen(1))
			ch := resp.Podcasts.Channels[0]
			Expect(ch.Id).To(Equal("ch-1"))
			Expect(ch.Title).To(Equal("Test Channel"))
			Expect(ch.Status).To(Equal("completed"))
			Expect(ch.Episode).To(HaveLen(1))
			Expect(ch.Episode[0].Id).To(Equal("ep-1"))
			Expect(ch.Episode[0].ChannelId).To(Equal("ch-1"))
			Expect(ch.Episode[0].Status).To(Equal("completed"))
		})

		It("getPodcasts with id returns a single channel", func() {
			resp, err := api.GetPodcasts(newPodcastRequest("getPodcasts", "id", "ch-1"))
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Podcasts.Channels).To(HaveLen(1))
			Expect(resp.Podcasts.Channels[0].Id).To(Equal("ch-1"))
			Expect(engine.lastChannelID).To(Equal("ch-1"))
		})

		It("getNewestPodcasts returns episodes", func() {
			resp, err := api.GetNewestPodcasts(newPodcastRequest("getNewestPodcasts", "count", "5"))
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.NewestPodcasts).ToNot(BeNil())
			Expect(resp.NewestPodcasts.Episodes).To(HaveLen(1))
			Expect(resp.NewestPodcasts.Episodes[0].Id).To(Equal("ep-1"))
		})

		It("getPodcastEpisode returns the episode", func() {
			engine.episode = &capabilities.PodcastEpisode{
				ID:        "ep-1",
				ChannelID: "ch-1",
				Title:     "Episode 1",
				Status:    capabilities.PodcastStatusCompleted,
			}
			resp, err := api.GetPodcastEpisode(newPodcastRequest("getPodcastEpisode", "id", "ep-1"))
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PodcastEpisode).ToNot(BeNil())
			Expect(resp.PodcastEpisode.Id).To(Equal("ep-1"))
			Expect(resp.PodcastEpisode.ChannelId).To(Equal("ch-1"))
		})

		It("createPodcastChannel delegates to the engine", func() {
			resp, err := api.CreatePodcastChannel(newPodcastRequest("createPodcastChannel", "url", "https://example.com/feed.xml"))
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("refreshPodcasts delegates to the engine", func() {
			resp, err := api.RefreshPodcasts(newPodcastRequest("refreshPodcasts"))
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("downloadPodcastEpisode delegates to the engine", func() {
			resp, err := api.DownloadPodcastEpisode(newPodcastRequest("downloadPodcastEpisode", "id", "ep-1"))
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(engine.lastEpisodeID).To(Equal("ep-1"))
		})

		It("deletePodcastChannel delegates to the engine", func() {
			resp, err := api.DeletePodcastChannel(newPodcastRequest("deletePodcastChannel", "id", "ch-1"))
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(engine.lastChannelID).To(Equal("ch-1"))
		})

		It("deletePodcastEpisode delegates to the engine", func() {
			resp, err := api.DeletePodcastEpisode(newPodcastRequest("deletePodcastEpisode", "id", "ep-1"))
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(engine.lastEpisodeID).To(Equal("ep-1"))
		})
	})
})
