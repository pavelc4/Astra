package meta

type ThreadsPhotoVariant struct {
	Resolution string `json:"resolution"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	URL        string `json:"url"`
}

type ThreadsPhoto struct {
	Index     int                   `json:"index"`
	Thumbnail *string               `json:"thumbnail,omitempty"`
	Variants  []ThreadsPhotoVariant `json:"variants"`
}

type ThreadsVideo struct {
	Index     int     `json:"index"`
	Thumbnail *string `json:"thumbnail,omitempty"`
	URL       string  `json:"url"`
	Format    string  `json:"format"`
}

type ThreadsResult struct {
	Platform   string         `json:"platform"`
	PhotoCount int            `json:"photoCount"`
	VideoCount int            `json:"videoCount"`
	Photos     []ThreadsPhoto `json:"photos"`
	Videos     []ThreadsVideo `json:"videos"`
}
