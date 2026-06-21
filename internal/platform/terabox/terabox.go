package terabox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

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

type teraboxFile struct {
	Name         string          `json:"name"`
	Size         string          `json:"size"`
	FileType     string          `json:"file_type"`
	DownloadURL  string          `json:"download_url"`
	ZipURL       string          `json:"zip_url"`
	ThumbnailURL string          `json:"thumbnail_url"`
	StreamURLs   json.RawMessage `json:"stream_urls"`
	SubtitleURL  string          `json:"subtitle_url"`
	Duration     string          `json:"duration"`
	Quality      string          `json:"quality"`
}

type teraboxData struct {
	Status string        `json:"status"`
	Files  []teraboxFile `json:"files"`
}

var nonceRe = regexp.MustCompile(`"nonce":"(.*?)"`)

func FetchData(ctx context.Context, teraboxURL string) (*Result, error) {
	nonce, err := fetchNonce(ctx)
	if err != nil {
		return nil, err
	}

	form := url.Values{
		"action": {"terabox_fetch"},
		"url":    {teraboxURL},
		"nonce":  {nonce},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://teradownloaderz.com/wp-admin/admin-ajax.php", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", "https://teradownloaderz.com")
	req.Header.Set("Referer", "https://teradownloaderz.com/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("Terabox fetch failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("Terabox response read failed")
	}

	var apiResp struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("Failed to parse Terabox API response: %s", err.Error()))
	}

	if !apiResp.Success {
		var errMsg string
		_ = json.Unmarshal(apiResp.Data, &errMsg)
		if errMsg == "" {
			errMsg = string(body)
		}
		return nil, errors.NewUpstream(fmt.Sprintf("Terabox API security/fetch failed: %s", errMsg))
	}

	var data teraboxData
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("Failed to parse Terabox data block: %s", err.Error()))
	}

	if data.Status != "success" {
		return nil, errors.NewUpstream("Terabox API returned non-success status")
	}

	var downloads []types.DownloadItem
	var title string
	var thumbnail string

	for _, file := range data.Files {
		if title == "" {
			title = file.Name
		}
		if thumbnail == "" {
			thumbnail = file.ThumbnailURL
		}

		mediaType := types.MediaVideo
		switch file.FileType {
		case "video":
			mediaType = types.MediaVideo
		case "image":
			mediaType = types.MediaImage
		case "audio":
			mediaType = types.MediaAudio
		}

		if file.DownloadURL != "" {
			downloads = append(downloads, types.DownloadItem{
				Label:   file.Name,
				URL:     file.DownloadURL,
				Type:    mediaType,
				Quality: file.Quality,
			})
		}

		if len(file.StreamURLs) > 0 {
			var streamMap map[string]string
			if err := json.Unmarshal(file.StreamURLs, &streamMap); err == nil {
				for q, url := range streamMap {
					if url != "" {
						downloads = append(downloads, types.DownloadItem{
							Label:   fmt.Sprintf("%s (Stream %s)", file.Name, q),
							URL:     url,
							Type:    types.MediaVideo,
							Quality: q,
						})
					}
				}
			} else {
				var streamStr string
				if err := json.Unmarshal(file.StreamURLs, &streamStr); err == nil && streamStr != "" {
					downloads = append(downloads, types.DownloadItem{
						Label:   fmt.Sprintf("%s (Stream)", file.Name),
						URL:     streamStr,
						Type:    types.MediaVideo,
						Quality: file.Quality,
					})
				}
			}
		}
	}

	return &Result{
		Platform:  "terabox",
		Title:     title,
		Thumbnail: thumbnail,
		Downloads: downloads,
	}, nil
}

func fetchNonce(ctx context.Context) (string, error) {
	bypassURL := fmt.Sprintf("https://teradownloaderz.com/?nowprocket=1&t=%d", time.Now().UnixNano())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bypassURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return "", errors.NewUpstream(fmt.Sprintf("Terabox nonce fetch failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", errors.NewUpstream("Terabox nonce HTML parse failed")
	}

	script := doc.Find("#jquery-core-js-extra").Text()
	if script == "" {
		return "", errors.NewUpstream("Nonce script not found")
	}

	m := nonceRe.FindStringSubmatch(script)
	if len(m) < 2 {
		return "", errors.NewUpstream("Nonce not found")
	}

	return m[1], nil
}
