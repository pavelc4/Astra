package capcut

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

	req, err := http.NewRequest(http.MethodPost, "https://www.genviral.io/api/tools/social-downloader", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://www.genviral.io")
	req.Header.Set("Referer", "https://www.genviral.io/tools/download/capcut")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("Capcut API request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("Capcut response read failed")
	}

	return &Result{Platform: "capcut", Raw: json.RawMessage(body)}, nil
}
