package linkedin

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

	req, err := http.NewRequest(http.MethodPost, "https://saywhat.ai/api/fetch-linkedin-page/", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://saywhat.ai/tools/linkedin-video-downloader/")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("LinkedIn API request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("LinkedIn response read failed")
	}

	return &Result{Platform: "linkedin", Raw: json.RawMessage(body)}, nil
}
