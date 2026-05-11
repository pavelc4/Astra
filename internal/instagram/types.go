package instagram

type MediaItem struct {
	Quality   string  `json:"quality"`
	Thumbnail *string `json:"thumbnail,omitempty"`
	URL       string  `json:"url"`
}
