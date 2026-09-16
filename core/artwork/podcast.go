package artwork

import (
	"context"
	"errors"
	"io"
	"net/url"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// PodcastCoverResolver looks up the remote cover image URL for a podcast channel
// (the channel id is the artwork id's ID part). It is satisfied by an adapter over
// the podcast engine, so this package does not depend on core/podcast directly.
type PodcastCoverResolver interface {
	PodcastCoverURL(ctx context.Context, channelID string) (string, error)
}

// noPodcastCover is a no-op resolver used when no podcast plugin is configured,
// so the artwork service can be constructed without a podcast engine.
type noPodcastCover struct{}

func (noPodcastCover) PodcastCoverURL(_ context.Context, _ string) (string, error) {
	return "", model.ErrNotFound
}

// servePodcast resolves a podcast channel's cover by fetching the channel's
// OriginalImageUrl (a remote URL supplied by the podcast plugin) and streaming it
// back, optionally resized. There is no database/state row for podcast art: it is
// always read through, keyed by the channel id (and the remote URL, so a feed that
// changes its image invalidates the resized cache).
func (s *service) servePodcast(ctx context.Context, artID model.ArtworkID, size int, square bool) (*Image, error) {
	if s.podcastCover == nil {
		return nil, ErrUnavailable
	}
	coverURL, err := s.podcastCover.PodcastCoverURL(ctx, artID.ID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, ErrUnavailable
		}
		return nil, err
	}
	if coverURL == "" {
		return nil, ErrUnavailable
	}
	remote, err := url.Parse(coverURL)
	if err != nil || remote.Scheme == "" || remote.Host == "" {
		return nil, ErrUnavailable
	}
	// The resized cache key embeds the remote URL so an image change invalidates it.
	key := artID.Kind.Prefix() + "|" + coverURL
	open := func() (io.ReadCloser, error) {
		rc, _, gerr := fromURL(ctx, remote)
		if gerr != nil {
			log.Warn(ctx, "Artwork: Could not fetch podcast cover", "url", coverURL, gerr)
			return nil, gerr
		}
		return rc, nil
	}
	img, err := s.serveSource(ctx, key, "", time.Now(), size, square, open)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, ErrUnavailable
	}
	return img, nil
}
