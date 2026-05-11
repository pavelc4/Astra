package soundcloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
)

type Result struct {
	Platform string          `json:"platform"`
	Raw      json.RawMessage `json:"raw"`
}

func FetchData(targetURL string) (*Result, error) {
	token := "8b6e170975d92939bb67d8db567f82e43fa2da91e00a84f258af77c1186c5e8a"
	hash := "aHR0cHM6Ly9zb3VuZGNsb3VkLmNvbS9zb21icnNvbmdzL3VuZHJlc3NlZA%3D%3D1043YWlvLWRs"
	payload := fmt.Sprintf("url=%s&token=%s&hash=%s", url.QueryEscape(targetURL), token, hash)

	req, err := http.NewRequest(http.MethodPost, "https://urlmp4.com/wp-json/aio-dl/video-data/", strings.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "pll_language=en")
	req.Header.Set("Referer", "https://urlmp4.com/en/soundcloud-downloader/")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("SoundCloud API request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("SoundCloud response read failed")
	}

	return &Result{Platform: "soundcloud", Raw: json.RawMessage(body)}, nil
}
