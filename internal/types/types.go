package types

type MediaType string

const (
	MediaVideo MediaType = "video"
	MediaAudio MediaType = "audio"
	MediaImage MediaType = "image"
	MediaSlide MediaType = "slide"
)

type DownloadItem struct {
	Label   string    `json:"label,omitempty"`
	URL     string    `json:"url"`
	Type    MediaType `json:"type"`
	Quality string    `json:"quality,omitempty"`
}
