package spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
)

type Result struct {
	Platform string          `json:"platform"`
	Raw      json.RawMessage `json:"raw"`
}

func FetchData(targetURL string) (*Result, error) {
	payload, _ := json.Marshal(map[string]string{"url": targetURL})

	req, err := http.NewRequest(http.MethodPost, "https://musicfab.io/api/spotify", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://musicfab.io/")
	req.Header.Set("Origin", "https://musicfab.io")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("MusicFab API request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, errors.NewUpstream(fmt.Sprintf("MusicFab API error: %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("MusicFab response read failed")
	}

	return &Result{Platform: "spotify", Raw: json.RawMessage(body)}, nil
}
