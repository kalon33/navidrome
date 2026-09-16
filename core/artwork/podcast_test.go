package artwork

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type fakePodcastCover struct {
	url    string
	err    error
	called string
}

func (f *fakePodcastCover) PodcastCoverURL(_ context.Context, channelID string) (string, error) {
	f.called = channelID
	if f.err != nil {
		return "", f.err
	}
	return f.url, nil
}

var _ = Describe("Podcast artwork", func() {
	var (
		ctx      context.Context
		svc      Artwork
		server   *httptest.Server
		cover    *fakePodcastCover
		imgBytes []byte
	)

	BeforeEach(func() {
		ctx = context.Background()
		img := image.NewRGBA(image.Rect(0, 0, 2, 2))
		img.Set(0, 0, color.RGBA{R: 255})
		buf := bytes.Buffer{}
		Expect(png.Encode(&buf, img)).To(Succeed())
		imgBytes = buf.Bytes()
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(imgBytes)
		}))
		cover = &fakePodcastCover{url: server.URL}
		conf.Server.CacheFolder = conf.NewDir(GinkgoT().TempDir())
		ds := &tests.MockDataStore{}
		ffm := tests.NewMockFFmpeg("")
		store := NewImageStore(GinkgoT().TempDir())
		imgCache := cache.NewFileCache("PodcastTest", "100MB", "images", 0,
			func(ctx context.Context, arg cache.Item) (io.Reader, error) {
				return arg.(artworkReader).Reader(ctx)
			})
		Eventually(func() bool { return imgCache.Available(ctx) }, 10*time.Second).Should(BeTrue())
		svc = NewArtworkWithPodcastCover(ds, imgCache, store, ffm, cover)
	})

	AfterEach(func() {
		server.Close()
	})

	DescribeTable("serves a fetched remote podcast cover", func(size int) {
		artID := model.NewArtworkID(model.KindPodcastArtwork, "ch-1", nil)
		img, err := svc.Get(ctx, artID, size, false)
		Expect(err).ToNot(HaveOccurred())
		Expect(img).ToNot(BeNil())
		Expect(img.Placeholder).To(BeFalse())
		Expect(cover.called).To(Equal("ch-1"))
		defer img.Close()
		Expect(io.ReadAll(img)).To(Equal(imgBytes))
	},
		Entry("full size", 0),
		Entry("resized", 40),
	)

	It("returns unavailable when the channel has no cover URL", func() {
		cover.url = ""
		artID := model.NewArtworkID(model.KindPodcastArtwork, "ch-1", nil)
		_, err := svc.Get(ctx, artID, 0, false)
		Expect(err).To(MatchError(ErrUnavailable))
	})

	It("returns unavailable when no resolver is configured", func() {
		ds := &tests.MockDataStore{}
		noPodCache := cache.NewFileCache("NoPod", "1MB", "images", 0,
			func(context.Context, cache.Item) (io.Reader, error) { return nil, nil })
		Eventually(func() bool { return noPodCache.Available(ctx) }, 10*time.Second).Should(BeTrue())
		svcNoPod := NewArtwork(ds, noPodCache, NewImageStore(GinkgoT().TempDir()), tests.NewMockFFmpeg(""))
		artID := model.NewArtworkID(model.KindPodcastArtwork, "ch-1", nil)
		_, err := svcNoPod.Get(ctx, artID, 0, false)
		Expect(err).To(MatchError(ErrUnavailable))
	})
})
