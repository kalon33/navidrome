package artwork

import (
	"github.com/google/wire"
)

var Set = wire.NewSet(
	NewArtworkWithPodcastCover,
	GetImageCache,
	NewWorker,
	GetImageStore,
	NewUploader,
)
