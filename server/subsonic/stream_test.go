package subsonic

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/navidrome/navidrome/plugins/capabilities"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeStreamPodcastEngine is a minimal test double for podcastsvc.Engine that
// only implements the pieces the stream handler needs (HasProvider, GetEpisode).
type fakeStreamPodcastEngine struct {
	hasProvider bool
	episode     *capabilities.PodcastEpisode
	getErr      error
	lastID      string
}

func (f *fakeStreamPodcastEngine) HasProvider() bool { return f.hasProvider }
func (f *fakeStreamPodcastEngine) GetEpisode(_ context.Context, id string) (*capabilities.PodcastEpisode, error) {
	f.lastID = id
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.episode, nil
}

// The rest of the Engine interface is not used by the stream handler.
func (f *fakeStreamPodcastEngine) GetChannels(context.Context, bool) ([]capabilities.PodcastChannel, error) {
	return nil, nil
}
func (f *fakeStreamPodcastEngine) GetChannel(context.Context, string, bool) (*capabilities.PodcastChannel, error) {
	return nil, nil
}
func (f *fakeStreamPodcastEngine) GetNewestEpisodes(context.Context, int) ([]capabilities.PodcastEpisode, error) {
	return nil, nil
}
func (f *fakeStreamPodcastEngine) CreateChannel(context.Context, string) (*capabilities.PodcastChannel, error) {
	return nil, nil
}
func (f *fakeStreamPodcastEngine) RefreshChannels(context.Context, []string) ([]string, error) {
	return nil, nil
}
func (f *fakeStreamPodcastEngine) DownloadEpisode(context.Context, string) (*capabilities.PodcastEpisode, error) {
	return nil, nil
}
func (f *fakeStreamPodcastEngine) DeleteChannel(context.Context, string) error { return nil }
func (f *fakeStreamPodcastEngine) DeleteEpisode(context.Context, string) error { return nil }

func newStreamRequest(method, endpoint string, params ...string) *http.Request {
	v := url.Values{}
	for i := 0; i < len(params); i += 2 {
		v.Set(params[i], params[i+1])
	}
	r := httptest.NewRequest(method, "/rest/"+endpoint+"?"+v.Encode(), nil)
	return r
}

var _ = Describe("Stream (podcast episodes)", func() {
	var api *Router
	var enclosure *httptest.Server
	var enclosureURL string
	var requestedPath string
	var rangeHeader string

	BeforeEach(func() {
		requestedPath = ""
		rangeHeader = ""
		enclosure = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			rangeHeader = r.Header.Get("Range")
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Header().Set("Accept-Ranges", "bytes")
			if rangeHeader != "" {
				w.Header().Set("Content-Range", "bytes 0-1023/2048")
				w.Header().Set("Content-Length", "1024")
				w.WriteHeader(http.StatusPartialContent)
			} else {
				w.Header().Set("Content-Length", "2048")
				w.WriteHeader(http.StatusOK)
			}
			_, _ = w.Write([]byte(strings.Repeat("x", 2048)))
		}))
		enclosureURL = enclosure.URL + "/episode.mp3"
		DeferCleanup(func() { enclosure.Close() })
	})

	Describe("isPodcastEpisodeID", func() {
		It("matches ep- prefixed ids", func() {
			Expect(isPodcastEpisodeID("ep-abc")).To(BeTrue())
		})

		It("does not match library media file ids", func() {
			Expect(isPodcastEpisodeID("mf-abc")).To(BeFalse())
			Expect(isPodcastEpisodeID("al-abc")).To(BeFalse())
			Expect(isPodcastEpisodeID("plainid")).To(BeFalse())
		})
	})

	Describe("isClientDisconnect", func() {
		It("detects broken pipe errors", func() {
			Expect(isClientDisconnect(errors.New("write tcp ...: write: broken pipe"))).To(BeTrue())
		})

		It("detects connection reset errors", func() {
			Expect(isClientDisconnect(errors.New("read tcp ...: read: connection reset by peer"))).To(BeTrue())
		})

		It("detects context cancellation", func() {
			Expect(isClientDisconnect(context.Canceled)).To(BeTrue())
		})

		It("does not match unrelated errors", func() {
			Expect(isClientDisconnect(errors.New("some other failure"))).To(BeFalse())
		})
	})

	Describe("streamPodcastEpisode", func() {
		BeforeEach(func() {
			engine := &fakeStreamPodcastEngine{
				hasProvider: true,
				episode: &capabilities.PodcastEpisode{
					ID:        "ep-1",
					StreamURL: enclosureURL,
					Duration:  120,
				},
			}
			api = &Router{podcast: engine}
		})

		It("proxies the enclosure content for an ep- id", func() {
			w := httptest.NewRecorder()
			r := newStreamRequest("GET", "stream", "id", "ep-1")

			_, err := api.Stream(w, r)
			Expect(err).ToNot(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("audio/mpeg"))
			Expect(w.Header().Get("Accept-Ranges")).To(Equal("bytes"))
			Expect(w.Header().Get("X-Content-Duration")).To(Equal("120"))
			Expect(w.Body.Len()).To(Equal(2048))
			Expect(requestedPath).To(Equal("/episode.mp3"))
		})

		It("forwards the Range header to the publisher", func() {
			w := httptest.NewRecorder()
			r := newStreamRequest("GET", "stream", "id", "ep-1")
			r.Header.Set("Range", "bytes=0-1023")

			_, err := api.Stream(w, r)
			Expect(err).ToNot(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusPartialContent))
			Expect(w.Header().Get("Content-Range")).To(Equal("bytes 0-1023/2048"))
			Expect(rangeHeader).To(Equal("bytes=0-1023"))
		})

		It("responds to HEAD requests without a body", func() {
			w := httptest.NewRecorder()
			r := newStreamRequest("HEAD", "stream", "id", "ep-1")

			_, err := api.Stream(w, r)
			Expect(err).ToNot(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Body.Len()).To(Equal(0))
		})

		It("returns data-not-found when no provider is configured", func() {
			api.podcast = &fakeStreamPodcastEngine{hasProvider: false}
			w := httptest.NewRecorder()
			r := newStreamRequest("GET", "stream", "id", "ep-1")

			_, err := api.Stream(w, r)
			Expect(err).To(HaveOccurred())
		})

		It("returns data-not-found when the episode has no stream URL", func() {
			engine := api.podcast.(*fakeStreamPodcastEngine)
			engine.episode = &capabilities.PodcastEpisode{ID: "ep-1", StreamURL: ""}
			w := httptest.NewRecorder()
			r := newStreamRequest("GET", "stream", "id", "ep-1")

			_, err := api.Stream(w, r)
			Expect(err).To(HaveOccurred())
		})

		It("returns data-not-found when the episode does not exist", func() {
			engine := api.podcast.(*fakeStreamPodcastEngine)
			engine.episode = nil
			w := httptest.NewRecorder()
			r := newStreamRequest("GET", "stream", "id", "ep-missing")

			_, err := api.Stream(w, r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Download (podcast episodes)", func() {
		BeforeEach(func() {
			engine := &fakeStreamPodcastEngine{
				hasProvider: true,
				episode: &capabilities.PodcastEpisode{
					ID:        "ep-1",
					Title:     "Episode One",
					StreamURL: enclosureURL,
					Suffix:    "mp3",
					Duration:  120,
				},
			}
			api = &Router{podcast: engine}
		})

		It("proxies the enclosure with an attachment disposition for download", func() {
			w := httptest.NewRecorder()
			r := newStreamRequest("GET", "download", "id", "ep-1")

			_, err := api.Download(w, r)
			Expect(err).ToNot(HaveOccurred())
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("audio/mpeg"))
			Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="Episode One.mp3"`))
			Expect(w.Body.Len()).To(Equal(2048))
		})

		It("falls back to the id and no suffix when the episode has no title/suffix", func() {
			engine := api.podcast.(*fakeStreamPodcastEngine)
			engine.episode = &capabilities.PodcastEpisode{ID: "ep-2", StreamURL: enclosureURL}
			w := httptest.NewRecorder()
			r := newStreamRequest("GET", "download", "id", "ep-2")

			_, err := api.Download(w, r)
			Expect(err).ToNot(HaveOccurred())
			Expect(w.Header().Get("Content-Disposition")).To(Equal(`attachment; filename="ep-2"`))
		})

		It("returns an error when no provider is configured", func() {
			api.podcast = &fakeStreamPodcastEngine{hasProvider: false}
			w := httptest.NewRecorder()
			r := newStreamRequest("GET", "download", "id", "ep-1")

			_, err := api.Download(w, r)
			Expect(err).To(HaveOccurred())
		})
	})
})
