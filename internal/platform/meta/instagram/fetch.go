package instagram

import (
	"context"

	"github.com/pavelc4/astra/internal/media"
)

// FetchData is the public download entry used by the HTTP handler. It runs the
// scraper and maps the internal MediaInfo onto the shared media.Media contract
// (meta shape: caption/owner/audio/photos/videos/raw).
//
// Items are appended photos-then-videos to match MediaInfo.Items
// (= append(Photos, Videos...) in graphql.go), so media's metaShape derives
// photos/videos/raw byte-identically to the pre-refactor InstagramResult.
func FetchData(ctx context.Context, url string) (*media.Media, error) {
	info, err := FetchMedia(ctx, url)
	if err != nil {
		return nil, err
	}

	items := make([]media.Item, 0, len(info.Photos)+len(info.Videos))
	for _, p := range info.Photos {
		items = append(items, toMediaItem(p, media.Image))
	}
	for _, v := range info.Videos {
		items = append(items, toMediaItem(v, media.Video))
	}

	return media.Meta("instagram", info.Caption, items,
		media.WithOwner(info.OwnerUser), media.WithAudio(info.AudioURL)), nil
}

func toMediaItem(m MediaItem, t media.Type) media.Item {
	it := media.Item{URL: m.URL, Quality: m.Quality, Type: t}
	if m.Thumbnail != nil {
		it.Thumbnail = *m.Thumbnail
	}
	return it
}

// FetchProfileByURL resolves a profile from either a profile URL (via username)
// or a media URL (via the owner of the post).
func FetchProfileByURL(ctx context.Context, rawURL string) (*UserProfile, error) {
	if username := ExtractUsername(rawURL); username != "" {
		return FetchProfile(ctx, username)
	}
	return FetchProfileFromMedia(ctx, rawURL)
}
