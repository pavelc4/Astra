package meta

import (
	"github.com/pavelc4/astra/internal/instagram"
)

type FacebookResult struct {
	Platform string               `json:"platform"`
	Caption  string               `json:"caption,omitempty"`
	Audio    string               `json:"audio,omitempty"`
	Photos   []instagram.MediaItem `json:"photos,omitempty"`
	Videos   []instagram.MediaItem `json:"videos,omitempty"`
	Raw      []instagram.MediaItem `json:"raw,omitempty"`
}

func FetchFacebookData(url string) (*FacebookResult, error) {
	info, err := instagram.FetchMedia(url)
	if err != nil {
		return nil, err
	}

	result := &FacebookResult{
		Platform: "facebook",
		Caption:  info.Caption,
		Audio:    info.AudioURL,
		Photos:   info.Photos,
		Videos:   info.Videos,
		Raw:      info.Items,
	}

	if result.Raw == nil {
		result.Raw = []instagram.MediaItem{}
	}

	return result, nil
}
