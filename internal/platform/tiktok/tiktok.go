package tiktok

import (
	"fmt"

	"github.com/heilkit/tt/tt"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/types"
)

type Result struct {
	Platform  string               `json:"platform"`
	Title     *string              `json:"title"`
	Thumbnail *string              `json:"thumbnail"`
	Downloads []types.DownloadItem `json:"downloads"`
}

func FetchData(videoURL string) (*Result, error) {
	post, err := tt.GetPost(videoURL, true)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("TikTok fetch failed: %s", err.Error()))
	}

	if post == nil {
		return nil, errors.NewUpstream("TikTok returned empty response")
	}

	var title *string
	if post.Title != "" {
		title = &post.Title
	}

	var downloads []types.DownloadItem

	if post.Hdplay != "" {
		downloads = append(downloads, types.DownloadItem{
			Label: "HD",
			URL:   post.Hdplay,
			Type:  types.MediaVideo,
		})
	}
	if post.Play != "" {
		downloads = append(downloads, types.DownloadItem{
			Label: "No Watermark",
			URL:   post.Play,
			Type:  types.MediaVideo,
		})
	}
	if post.Wmplay != "" {
		downloads = append(downloads, types.DownloadItem{
			Label: "With Watermark",
			URL:   post.Wmplay,
			Type:  types.MediaVideo,
		})
	}
	if post.Music != "" {
		downloads = append(downloads, types.DownloadItem{
			Label: "Audio",
			URL:   post.Music,
			Type:  types.MediaAudio,
		})
	}
	for _, img := range post.Images {
		if img != "" {
			downloads = append(downloads, types.DownloadItem{
				URL:  img,
				Type: types.MediaImage,
			})
		}
	}

	return &Result{
		Platform: "tiktok",
		Title:    title,
		Downloads: downloads,
	}, nil
}
