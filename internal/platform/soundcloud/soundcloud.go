package soundcloud

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
	"github.com/pavelc4/astra/internal/types"
)

type Result struct {
	Platform  string               `json:"platform"`
	Title     string               `json:"title"`
	Thumbnail string               `json:"thumbnail,omitempty"`
	Downloads []types.DownloadItem `json:"downloads"`
}

type RawMedia struct {
	URL            string `json:"url"`
	Quality        string `json:"quality"`
	Extension      string `json:"extension"`
	Size           int64  `json:"size"`
	AudioAvailable bool   `json:"audioAvailable"`
}

type RawResponse struct {
	Title     string     `json:"title"`
	Thumbnail string     `json:"thumbnail"`
	Medias    []RawMedia `json:"medias"`
}

func FetchData(ctx context.Context, targetURL string) (*Result, error) {
	token := "8b6e170975d92939bb67d8db567f82e43fa2da91e00a84f258af77c1186c5e8a"
	encodedURL := base64.StdEncoding.EncodeToString([]byte(targetURL))
	hash := url.QueryEscape(encodedURL + "1043YWlvLWRs")
	payload := fmt.Sprintf("url=%s&token=%s&hash=%s", url.QueryEscape(targetURL), token, hash)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://urlmp4.com/wp-json/aio-dl/video-data/", strings.NewReader(payload))
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

	var raw RawResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, errors.NewUpstream("failed to parse SoundCloud response")
	}

	var downloads []types.DownloadItem
	for _, m := range raw.Medias {
		if strings.HasSuffix(m.URL, ".json") {
			continue
		}

		mediaType := types.MediaAudio
		if !m.AudioAvailable {
			mediaType = types.MediaVideo
		}

		downloads = append(downloads, types.DownloadItem{
			Label:   fmt.Sprintf("Audio %s (%s)", m.Quality, m.Extension),
			URL:     m.URL,
			Type:    mediaType,
			Quality: m.Quality,
		})
	}

	if len(downloads) == 0 {
		return nil, errors.NewUpstream("no downloadable media found in SoundCloud post")
	}

	return &Result{
		Platform:  "soundcloud",
		Title:     raw.Title,
		Thumbnail: raw.Thumbnail,
		Downloads: downloads,
	}, nil
}
