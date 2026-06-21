package meta

import (
	"context"

	"github.com/pavelc4/astra/internal/instagram"
)

type InstagramResult struct {
	Platform    string               `json:"platform"`
	Caption     string               `json:"caption,omitempty"`
	Owner       string               `json:"owner,omitempty"`
	Audio       string               `json:"audio,omitempty"`
	Photos      []instagram.MediaItem `json:"photos,omitempty"`
	Videos      []instagram.MediaItem `json:"videos,omitempty"`
	Raw         []instagram.MediaItem `json:"raw,omitempty"`
}

func FetchInstagramProfile(ctx context.Context, rawURL string) (*instagram.UserProfile, error) {
	if username := instagram.ExtractUsername(rawURL); username != "" {
		return instagram.FetchProfile(ctx, username)
	}

	return instagram.FetchProfileFromMedia(ctx, rawURL)
}

func FetchInstagramData(ctx context.Context, url string) (*InstagramResult, error) {
	info, err := instagram.FetchMedia(ctx, url)
	if err != nil {
		return nil, err
	}

	result := &InstagramResult{
		Platform: "instagram",
		Caption:  info.Caption,
		Owner:    info.OwnerUser,
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
