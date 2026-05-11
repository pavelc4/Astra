package meta

type InstagramResult struct {
	Platform string         `json:"platform"`
	Raw      []SnapSaveItem `json:"raw"`
}

func FetchInstagramData(url string) (*InstagramResult, error) {
	items, err := fetchSnapsave(url)
	if err != nil {
		return nil, err
	}
	return &InstagramResult{Platform: "instagram", Raw: items}, nil
}
