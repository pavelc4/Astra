package types

import "github.com/pavelc4/astra/internal/media"

// Media types now live in internal/media; these aliases keep platform packages
// that still import types compiling during the migration to media.Media.
//
// ponytail: transitional shim — removed once every platform imports media
// directly (tracked in the meta-merge PR).
type MediaType = media.Type

const (
	MediaVideo = media.Video
	MediaAudio = media.Audio
	MediaImage = media.Image
	MediaSlide = media.Slide
)

type DownloadItem = media.Item
