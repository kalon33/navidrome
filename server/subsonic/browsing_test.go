package subsonic

import (
	"context"
	"fmt"
	"net/http/httptest"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/capabilities"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func contextWithUser(ctx context.Context, userID string, libraryIDs ...int) context.Context {
	libraries := make([]model.Library, len(libraryIDs))
	for i, id := range libraryIDs {
		libraries[i] = model.Library{ID: id, Name: fmt.Sprintf("Test Library %d", id), Path: fmt.Sprintf("/music/library%d", id)}
	}
	user := model.User{
		ID:        userID,
		Libraries: libraries,
	}
	return request.WithUser(ctx, user)
}

var _ = Describe("Browsing", func() {
	var api *Router
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		auth.Init(ds)
		api = &Router{ds: ds}
		ctx = context.Background()
	})

	Describe("GetMusicFolders", func() {
		It("should return all libraries the user has access", func() {
			// Create mock user with libraries
			ctx := contextWithUser(ctx, "user-id", 1, 2, 3)

			// Create request
			r := httptest.NewRequest("GET", "/rest/getMusicFolders", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetMusicFolders(r)

			// Verify results
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.MusicFolders).ToNot(BeNil())
			Expect(response.MusicFolders.Folders).To(HaveLen(3))
			Expect(response.MusicFolders.Folders[0].Name).To(Equal("Test Library 1"))
			Expect(response.MusicFolders.Folders[1].Name).To(Equal("Test Library 2"))
			Expect(response.MusicFolders.Folders[2].Name).To(Equal("Test Library 3"))
		})
	})

	Describe("GetIndexes", func() {
		It("should validate user access to the specified musicFolderId", func() {
			// Create mock user with access to library 1 only
			ctx = contextWithUser(ctx, "user-id", 1)

			// Create request with musicFolderId=2 (not accessible)
			r := httptest.NewRequest("GET", "/rest/getIndexes?musicFolderId=2", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetIndexes(r)

			// Should return error due to lack of access
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("should default to first accessible library when no musicFolderId specified", func() {
			// Create mock user with access to libraries 2 and 3
			ctx = contextWithUser(ctx, "user-id", 2, 3)

			// Setup minimal mock library data for working tests
			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(model.Libraries{
				{ID: 2, Name: "Test Library 2", Path: "/music/library2"},
				{ID: 3, Name: "Test Library 3", Path: "/music/library3"},
			})

			// Setup mock artist data
			mockArtistRepo := ds.Artist(ctx).(*tests.MockArtistRepo)
			mockArtistRepo.SetData(model.Artists{
				{ID: "1", Name: "Test Artist 1"},
				{ID: "2", Name: "Test Artist 2"},
			})

			// Create request without musicFolderId
			r := httptest.NewRequest("GET", "/rest/getIndexes", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetIndexes(r)

			// Should succeed and use first accessible library (2)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.Indexes).ToNot(BeNil())
		})
	})

	Describe("GetArtists", func() {
		It("should validate user access to the specified musicFolderId", func() {
			// Create mock user with access to library 1 only
			ctx = contextWithUser(ctx, "user-id", 1)

			// Create request with musicFolderId=3 (not accessible)
			r := httptest.NewRequest("GET", "/rest/getArtists?musicFolderId=3", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetArtists(r)

			// Should return error due to lack of access
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("should default to first accessible library when no musicFolderId specified", func() {
			// Create mock user with access to libraries 1 and 2
			ctx = contextWithUser(ctx, "user-id", 1, 2)

			// Setup minimal mock library data for working tests
			mockLibRepo := ds.Library(ctx).(*tests.MockLibraryRepo)
			mockLibRepo.SetData(model.Libraries{
				{ID: 1, Name: "Test Library 1", Path: "/music/library1"},
				{ID: 2, Name: "Test Library 2", Path: "/music/library2"},
			})

			// Setup mock artist data
			mockArtistRepo := ds.Artist(ctx).(*tests.MockArtistRepo)
			mockArtistRepo.SetData(model.Artists{
				{ID: "1", Name: "Test Artist 1"},
				{ID: "2", Name: "Test Artist 2"},
			})

			// Create request without musicFolderId
			r := httptest.NewRequest("GET", "/rest/getArtists", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetArtists(r)

			// Should succeed and use first accessible library (1)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.Artist).ToNot(BeNil())
		})
	})

	Describe("GetAlbumInfo", func() {
		It("emits image URLs when the album artwork is unresolved", func() {
			api.provider = &fakeInfoProvider{album: &model.Album{ID: "al-1"}}
			r := httptest.NewRequest("GET", "/rest/getAlbumInfo?id=al-1", nil)
			resp, err := api.GetAlbumInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumInfo.SmallImageUrl).ToNot(BeEmpty())
			Expect(resp.AlbumInfo.LargeImageUrl).ToNot(BeEmpty())
		})
		It("omits image URLs when the album artwork is known absent", func() {
			api.provider = &fakeInfoProvider{album: &model.Album{ID: "al-1", ItemImage: model.ItemImage{ImageAbsent: true}}}
			r := httptest.NewRequest("GET", "/rest/getAlbumInfo?id=al-1", nil)
			resp, err := api.GetAlbumInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumInfo.SmallImageUrl).To(BeEmpty())
			Expect(resp.AlbumInfo.MediumImageUrl).To(BeEmpty())
			Expect(resp.AlbumInfo.LargeImageUrl).To(BeEmpty())
		})
	})

	Describe("GetSimilarSongs", func() {
		It("returns an empty result for podcast episode ids instead of an error", func() {
			api = &Router{ds: ds}
			r := httptest.NewRequest("GET", "/rest/getSimilarSongs?id=ep-1&count=10", nil)
			resp, err := api.GetSimilarSongs(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.SimilarSongs).ToNot(BeNil())
			Expect(resp.SimilarSongs.Song).To(BeEmpty())
		})
	})

	Describe("GetSong", func() {
		It("returns podcast episode metadata for ep- ids", func() {
			api = &Router{ds: ds, podcast: &fakePodcastEngine{
				hasProvider: true,
				episode: &capabilities.PodcastEpisode{
					ID:        "ep-1",
					Title:     "Episode One",
					ChannelID: "ch-1",
					Status:    capabilities.PodcastStatusCompleted,
					StreamURL: "https://example.com/ep1.mp3",
					Suffix:    "mp3",
					Duration:  120,
				},
			}}
			r := httptest.NewRequest("GET", "/rest/getSong?id=ep-1", nil)
			resp, err := api.GetSong(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Song).ToNot(BeNil())
			Expect(resp.Song.Id).To(Equal("ep-1"))
			Expect(resp.Song.Title).To(Equal("Episode One"))
			Expect(resp.Song.Type).To(Equal("podcast"))
			Expect(resp.Song.Duration).To(Equal(int32(120)))
		})

		It("returns not-found when no podcast provider is configured for ep- ids", func() {
			api = &Router{ds: ds}
			r := httptest.NewRequest("GET", "/rest/getSong?id=ep-missing", nil)
			_, err := api.GetSong(r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetAlbum / GetArtist with empty id", func() {
		It("returns not-found for an empty album id without an error log", func() {
			api = &Router{ds: ds}
			r := httptest.NewRequest("GET", "/rest/getAlbum?id=", nil)
			_, err := api.GetAlbum(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns not-found for an empty artist id without an error log", func() {
			api = &Router{ds: ds}
			r := httptest.NewRequest("GET", "/rest/getArtist?id=", nil)
			_, err := api.GetArtist(r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetArtistInfo", func() {
		It("emits image URLs when the artist artwork is unresolved", func() {
			api.provider = &fakeInfoProvider{artist: &model.Artist{ID: "ar-1"}}
			r := httptest.NewRequest("GET", "/rest/getArtistInfo?id=ar-1", nil)
			resp, err := api.GetArtistInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ArtistInfo.SmallImageUrl).ToNot(BeEmpty())
		})
		It("omits image URLs when the artist artwork is known absent", func() {
			api.provider = &fakeInfoProvider{artist: &model.Artist{ID: "ar-1", ItemImage: model.ItemImage{ImageAbsent: true}}}
			r := httptest.NewRequest("GET", "/rest/getArtistInfo?id=ar-1", nil)
			resp, err := api.GetArtistInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ArtistInfo.SmallImageUrl).To(BeEmpty())
			Expect(resp.ArtistInfo.MediumImageUrl).To(BeEmpty())
			Expect(resp.ArtistInfo.LargeImageUrl).To(BeEmpty())
		})
	})
})

type fakeInfoProvider struct {
	external.Provider
	album  *model.Album
	artist *model.Artist
}

func (f *fakeInfoProvider) UpdateAlbumInfo(context.Context, string) (*model.Album, error) {
	return f.album, nil
}

func (f *fakeInfoProvider) UpdateArtistInfo(context.Context, string, int, bool) (*model.Artist, error) {
	return f.artist, nil
}
