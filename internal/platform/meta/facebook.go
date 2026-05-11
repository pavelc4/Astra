package meta

import (
	"github.com/pavelc4/astra/internal/instagram"
)

type FacebookResult struct {
	Platform string                `json:"platform"`
	Raw      []instagram.MediaItem `json:"raw"`
}

func FetchFacebookData(url string) (*FacebookResult, error) {
	items, err := instagram.FetchMedia(url)
	if err != nil {
		return nil, err
	}
	return &FacebookResult{Platform: "facebook", Raw: items}, nil
}
