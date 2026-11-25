package subsonic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/plugins/capabilities"
	"github.com/navidrome/navidrome/server/subsonic/responses"
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

	// putArtistPerLibrary stores album artist ar-N in each library N, all granted to user-id.
	putArtistPerLibrary := func() {
		GinkgoHelper()
		putUsers(ctx, ds, model.User{ID: "user-id", UserName: "user-id"})
		for _, id := range persistencetest.SetLibraries(ctx, ds, "/music/library1", "/music/library2", "/music/library3") {
			putAlbumArtists(ctx, ds, id, model.Artist{ID: fmt.Sprintf("ar-%d", id), Name: fmt.Sprintf("Test Artist %d", id)})
		}
	}
	indexedArtistIDs := func(indexes []responses.Index) []string {
		var ids []string
		for _, idx := range indexes {
			for _, a := range idx.Artists {
				ids = append(ids, a.Id)
			}
		}
		return ids
	}
	indexedArtistID3IDs := func(indexes []responses.IndexID3) []string {
		var ids []string
		for _, idx := range indexes {
			for _, a := range idx.Artists {
				ids = append(ids, a.Id)
			}
		}
		return ids
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		auth.Init(ds)
		api = &Router{ds: ds}
	})

	Describe("GetMusicFolders", func() {
		It("should return all libraries the user has access", func() {
			// Create mock user with libraries
			ctx := contextWithUser(ctx, "user-id", 1, 2, 3)

			// Create request
			r := httptest.NewRequest(http.MethodGet, "/rest/getMusicFolders", nil)
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
			putArtistPerLibrary()
			// Create mock user with access to library 1 only
			ctx = contextWithUser(ctx, "user-id", 1)

			// Create request with musicFolderId=2 (not accessible)
			r := httptest.NewRequest(http.MethodGet, "/rest/getIndexes?musicFolderId=2", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetIndexes(r)

			// Should return error due to lack of access
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("should default to first accessible library when no musicFolderId specified", func() {
			putArtistPerLibrary()
			// Create mock user with access to libraries 2 and 3
			ctx = contextWithUser(ctx, "user-id", 2, 3)

			// Create request without musicFolderId
			r := httptest.NewRequest(http.MethodGet, "/rest/getIndexes", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetIndexes(r)

			// Should succeed and use first accessible library (2)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.Indexes).ToNot(BeNil())
			Expect(indexedArtistIDs(response.Indexes.Index)).To(ConsistOf("ar-2", "ar-3"))
		})
	})

	Describe("GetArtists", func() {
		It("should validate user access to the specified musicFolderId", func() {
			putArtistPerLibrary()
			// Create mock user with access to library 1 only
			ctx = contextWithUser(ctx, "user-id", 1)

			// Create request with musicFolderId=3 (not accessible)
			r := httptest.NewRequest(http.MethodGet, "/rest/getArtists?musicFolderId=3", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetArtists(r)

			// Should return error due to lack of access
			Expect(err).To(HaveOccurred())
			Expect(response).To(BeNil())
		})

		It("should default to first accessible library when no musicFolderId specified", func() {
			putArtistPerLibrary()
			// Create mock user with access to libraries 1 and 2
			ctx = contextWithUser(ctx, "user-id", 1, 2)

			// Create request without musicFolderId
			r := httptest.NewRequest(http.MethodGet, "/rest/getArtists", nil)
			r = r.WithContext(ctx)

			// Call endpoint
			response, err := api.GetArtists(r)

			// Should succeed and use first accessible library (1)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())
			Expect(response.Artist).ToNot(BeNil())
			Expect(indexedArtistID3IDs(response.Artist.Index)).To(ConsistOf("ar-1", "ar-2"))
		})
	})

	Describe("GetAlbumInfo", func() {
		It("emits image URLs when the album artwork is unresolved", func() {
			api.provider = &fakeInfoProvider{album: &model.Album{ID: "al-1"}}
			r := httptest.NewRequest(http.MethodGet, "/rest/getAlbumInfo?id=al-1", nil)
			resp, err := api.GetAlbumInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumInfo.SmallImageUrl).ToNot(BeEmpty())
			Expect(resp.AlbumInfo.LargeImageUrl).ToNot(BeEmpty())
		})
		It("omits image URLs when the album artwork is known absent", func() {
			api.provider = &fakeInfoProvider{album: &model.Album{ID: "al-1", ImageAbsent: true}}
			r := httptest.NewRequest(http.MethodGet, "/rest/getAlbumInfo?id=al-1", nil)
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
			r := httptest.NewRequest(http.MethodGet, "/rest/getSimilarSongs?id=ep-1&count=10", nil)
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
				channel: &capabilities.PodcastChannel{
					ID:     "ch-1",
					Title:  "The Show",
					Status: capabilities.PodcastStatusCompleted,
				},
			}}
			r := httptest.NewRequest(http.MethodGet, "/rest/getSong?id=ep-1", nil)
			resp, err := api.GetSong(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Song).ToNot(BeNil())
			Expect(resp.Song.Id).To(Equal("ep-1"))
			Expect(resp.Song.Title).To(Equal("Episode One"))
			Expect(resp.Song.Type).To(Equal("podcast"))
			Expect(resp.Song.Duration).To(Equal(int32(120)))
			// The feed name is exposed as the episode's artist so players display it.
			Expect(resp.Song.Artist).To(Equal("The Show"))
			Expect(resp.Song.Album).To(Equal("The Show"))
		})

		It("returns not-found when no podcast provider is configured for ep- ids", func() {
			api = &Router{ds: ds}
			r := httptest.NewRequest(http.MethodGet, "/rest/getSong?id=ep-missing", nil)
			_, err := api.GetSong(r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetAlbum / GetArtist with empty id", func() {
		It("returns not-found for an empty album id without an error log", func() {
			api = &Router{ds: ds}
			r := httptest.NewRequest(http.MethodGet, "/rest/getAlbum?id=", nil)
			_, err := api.GetAlbum(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns not-found for an empty artist id without an error log", func() {
			api = &Router{ds: ds}
			r := httptest.NewRequest(http.MethodGet, "/rest/getArtist?id=", nil)
			_, err := api.GetArtist(r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("stripTranscodeSuffix", func() {
		It("strips a -<format>.<ext> suffix", func() {
			Expect(stripTranscodeSuffix("9hpV710cCTSLMF7H0XpYAo-raw.flc")).To(Equal("9hpV710cCTSLMF7H0XpYAo"))
			Expect(stripTranscodeSuffix("abc123-mp3.mp3")).To(Equal("abc123"))
		})

		It("leaves a bare id unchanged", func() {
			Expect(stripTranscodeSuffix("9hpV710cCTSLMF7H0XpYAo")).To(Equal("9hpV710cCTSLMF7H0XpYAo"))
		})

		It("leaves an id with a dash but no format.ext suffix unchanged", func() {
			Expect(stripTranscodeSuffix("ep-1")).To(Equal("ep-1"))
			Expect(stripTranscodeSuffix("some-id")).To(Equal("some-id"))
		})

		It("does not strip a suffix with no extension", func() {
			Expect(stripTranscodeSuffix("id-raw")).To(Equal("id-raw"))
		})
	})

	Describe("GetArtistInfo", func() {
		It("emits image URLs when the artist artwork is unresolved", func() {
			api.provider = &fakeInfoProvider{artist: &model.Artist{ID: "ar-1"}}
			r := httptest.NewRequest(http.MethodGet, "/rest/getArtistInfo?id=ar-1", nil)
			resp, err := api.GetArtistInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ArtistInfo.SmallImageUrl).ToNot(BeEmpty())
		})
		It("omits image URLs when the artist artwork is known absent", func() {
			api.provider = &fakeInfoProvider{artist: &model.Artist{ID: "ar-1", ImageAbsent: true}}
			r := httptest.NewRequest(http.MethodGet, "/rest/getArtistInfo?id=ar-1", nil)
			resp, err := api.GetArtistInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ArtistInfo.SmallImageUrl).To(BeEmpty())
			Expect(resp.ArtistInfo.MediumImageUrl).To(BeEmpty())
			Expect(resp.ArtistInfo.LargeImageUrl).To(BeEmpty())
		})
	})

	Describe("similar artists not in the library", func() {
		BeforeEach(func() {
			api.provider = &fakeInfoProvider{artist: &model.Artist{ID: "ar-1", SimilarArtists: model.Artists{
				{ID: "ar-2", Name: "In Library"},
				{Name: "Not In Library"},
			}}}
		})

		It("GetArtistInfo omits the artistImageUrl only for the artist not in the library", func() {
			r := httptest.NewRequest(http.MethodGet, "/rest/getArtistInfo?id=ar-1&includeNotPresent=true", nil)
			resp, err := api.GetArtistInfo(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ArtistInfo.SimilarArtist).To(HaveLen(2))
			Expect(resp.ArtistInfo.SimilarArtist[0].ArtistImageUrl).ToNot(BeEmpty())
			Expect(resp.ArtistInfo.SimilarArtist[1].Id).To(Equal("-1"))
			Expect(resp.ArtistInfo.SimilarArtist[1].CoverArt).To(BeEmpty())
			Expect(resp.ArtistInfo.SimilarArtist[1].ArtistImageUrl).To(BeEmpty())
		})

		It("GetArtistInfo2 omits the artistImageUrl only for the artist not in the library", func() {
			r := httptest.NewRequest(http.MethodGet, "/rest/getArtistInfo2?id=ar-1&includeNotPresent=true", nil)
			resp, err := api.GetArtistInfo2(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ArtistInfo2.SimilarArtist).To(HaveLen(2))
			Expect(resp.ArtistInfo2.SimilarArtist[0].ArtistImageUrl).ToNot(BeEmpty())
			Expect(resp.ArtistInfo2.SimilarArtist[1].Id).To(Equal("-1"))
			Expect(resp.ArtistInfo2.SimilarArtist[1].CoverArt).To(BeEmpty())
			Expect(resp.ArtistInfo2.SimilarArtist[1].ArtistImageUrl).To(BeEmpty())
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
