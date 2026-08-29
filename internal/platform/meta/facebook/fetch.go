package facebook

import (
	"context"

	"github.com/pavelc4/astra/internal/media"
)

// FetchData is the public entry point used by the HTTP handler. It runs the
// scraper (FetchMedia) and maps the internal MediaInfo onto the shared
// media.Media contract (meta shape: caption/duration/photos/videos/raw).
//
// Videos are appended before photos so media.Media's raw ordering matches the
// pre-refactor response byte-for-byte. Video items carry the post thumbnail;
// photos do not — same as before.
func FetchData(ctx context.Context, url string) (*media.Media, error) {
	info, err := FetchMedia(ctx, url)
	if err != nil {
		return nil, err
	}

	items := make([]media.Item, 0, len(info.Videos)+len(info.Photos))
	for _, v := range info.Videos {
		it := media.Item{URL: v.URL, Type: media.Video, Quality: v.Quality}
		if info.Thumbnail != nil {
			it.Thumbnail = *info.Thumbnail
		}
		items = append(items, it)
	}
	for _, p := range info.Photos {
		items = append(items, media.Item{URL: p.URL, Type: media.Image, Quality: p.Quality})
	}

	return media.Meta("facebook", info.Caption, items, media.WithDuration(info.Duration)), nil
}
