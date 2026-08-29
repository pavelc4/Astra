package facebook

type MediaItem struct {
	Quality   string  `json:"quality"`
	Thumbnail *string `json:"thumbnail,omitempty"`
	URL       string  `json:"url"`
}

type MediaInfo struct {
	Caption   string      `json:"caption,omitempty"`
	Thumbnail *string     `json:"thumbnail,omitempty"`
	Duration  string      `json:"duration,omitempty"`
	Photos    []MediaItem `json:"photos,omitempty"`
	Videos    []MediaItem `json:"videos,omitempty"`
}

type graphQLResponse struct {
	Data   *graphQLData   `json:"data,omitempty"`
	Errors []graphQLError `json:"errors,omitempty"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLData struct {
	Node *graphQLNode `json:"node,omitempty"`
}

type graphQLNode struct {
	Typename    string            `json:"__typename"`
	PlayableURL string            `json:"playable_url,omitempty"`
	PlayableHD  string            `json:"playable_url_quality_hd,omitempty"`
	BrowserHD   string            `json:"browser_native_hd_url,omitempty"`
	BrowserSD   string            `json:"browser_native_sd_url,omitempty"`
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	DurationMs  int               `json:"playable_duration_in_ms,omitempty"`
	Thumbnail   *graphQLThumbnail `json:"preferred_thumbnail,omitempty"`
	ID          string            `json:"id,omitempty"`
}

type graphQLThumbnail struct {
	Image *graphQLImage `json:"image,omitempty"`
}

type graphQLImage struct {
	URI string `json:"uri,omitempty"`
}
