package meta

type FacebookResult struct {
	Platform string         `json:"platform"`
	Raw      []SnapSaveItem `json:"raw"`
}

func FetchFacebookData(url string) (*FacebookResult, error) {
	items, err := fetchSnapsave(url)
	if err != nil {
		return nil, err
	}
	return &FacebookResult{Platform: "facebook", Raw: items}, nil
}
