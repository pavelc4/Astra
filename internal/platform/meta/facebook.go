package meta

import (
	"github.com/pavelc4/astra/internal/facebook"
	"github.com/pavelc4/astra/internal/instagram"
)

type FacebookResult struct {
	Platform  string                `json:"platform"`
	Caption   string                `json:"caption,omitempty"`
	Duration  string                `json:"duration,omitempty"`
	Audio     string                `json:"audio,omitempty"`
	Photos    []instagram.MediaItem `json:"photos,omitempty"`
	Videos    []instagram.MediaItem `json:"videos,omitempty"`
	Raw       []instagram.MediaItem `json:"raw,omitempty"`
}

func FetchFacebookData(url string) (*FacebookResult, error) {
	info, err := facebook.FetchMedia(url)
	if err != nil {
		return nil, err
	}

	result := &FacebookResult{
		Platform: "facebook",
		Caption:  info.Caption,
		Duration: info.Duration,
	}

	for _, v := range info.Videos {
		item := instagram.MediaItem{URL: v.URL, Quality: v.Quality, Thumbnail: info.Thumbnail}
		result.Videos = append(result.Videos, item)
		result.Raw = append(result.Raw, item)
	}

	for _, p := range info.Photos {
		item := instagram.MediaItem{URL: p.URL}
		result.Photos = append(result.Photos, item)
		result.Raw = append(result.Raw, item)
	}

	if result.Raw == nil {
		result.Raw = []instagram.MediaItem{}
	}

	return result, nil
}
