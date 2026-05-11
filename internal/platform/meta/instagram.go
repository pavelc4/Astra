package meta

import (
	"github.com/pavelc4/astra/internal/instagram"
)

type InstagramResult struct {
	Platform string              `json:"platform"`
	Raw      []instagram.MediaItem `json:"raw"`
}

func FetchInstagramData(url string) (*InstagramResult, error) {
	items, err := instagram.FetchMedia(url)
	if err != nil {
		return nil, err
	}
	return &InstagramResult{Platform: "instagram", Raw: items}, nil
}
