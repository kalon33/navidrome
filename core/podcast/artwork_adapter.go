package podcast

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/model"
)

// PodcastCoverURLAdapter exposes the channel cover URL resolution as the
// artwork.PodcastCoverResolver interface, without core/artwork depending on
// core/podcast. It is implemented here (next to the Engine) and bound via wire.
type PodcastCoverURLAdapter struct {
	engine Engine
}

// NewPodcastCoverURLAdapter wraps the given podcast engine so the artwork
// service can resolve podcast channel covers.
func NewPodcastCoverURLAdapter(engine Engine) *PodcastCoverURLAdapter {
	return &PodcastCoverURLAdapter{engine: engine}
}

// PodcastCoverURL returns the channel's OriginalImageUrl for the given channel
// id, or model.ErrNotFound when the channel (or a podcast provider) is absent.
func (a *PodcastCoverURLAdapter) PodcastCoverURL(ctx context.Context, channelID string) (string, error) {
	if a.engine == nil || !a.engine.HasProvider() {
		return "", model.ErrNotFound
	}
	ch, err := a.engine.GetChannel(ctx, channelID, false)
	if err != nil {
		if errors.Is(err, ErrNoProvider) {
			return "", model.ErrNotFound
		}
		return "", err
	}
	if ch == nil {
		return "", model.ErrNotFound
	}
	return ch.OriginalImageUrl, nil
}
