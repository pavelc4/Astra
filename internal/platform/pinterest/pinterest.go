package pinterest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
)

type DownloadItem struct {
	Quality string `json:"quality"`
	Format  string `json:"format"`
	URL     string `json:"url"`
}

type Result struct {
	Platform  string         `json:"platform"`
	Title     string         `json:"title"`
	Thumbnail string         `json:"thumbnail,omitempty"`
	Downloads []DownloadItem `json:"downloads"`
}

// pinterestClient is a custom http.Client configured to follow redirects for pin.it URLs.
var pinterestClient = &http.Client{
	Timeout: 10 * time.Second,
}

func FetchData(targetURL string) (*Result, error) {
	// 1. Resolve final URL to extract Pin ID
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := pinterestClient.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("failed to resolve URL: %s", err.Error()))
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	re := regexp.MustCompile(`/pin/(\d+)`)
	matches := re.FindStringSubmatch(finalURL)
	if len(matches) < 2 {
		return nil, errors.NewValidation(fmt.Sprintf("could not extract Pin ID from URL: %s", finalURL))
	}
	pinID := matches[1]

	// 2. Fetch detailed metadata from Pinterest internal resource API
	apiURL := "https://www.pinterest.com/resource/PinResource/get/"
	dataParam, err := json.Marshal(map[string]interface{}{
		"options": map[string]interface{}{
			"id":            pinID,
			"field_set_key": "detailed",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request options: %w", err)
	}

	params := url.Values{}
	params.Set("data", string(dataParam))
	fullAPIURL := apiURL + "?" + params.Encode()

	apiReq, err := http.NewRequest(http.MethodGet, fullAPIURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create API request: %w", err)
	}

	apiReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	apiReq.Header.Set("X-Pinterest-PWS-Handler", "www/[username].js")
	apiReq.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	apiReq.Header.Set("X-Requested-With", "XMLHttpRequest")

	apiResp, err := httpclient.Client.Do(apiReq)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("Pinterest API request failed: %s", err.Error()))
	}
	defer apiResp.Body.Close()

	if apiResp.StatusCode != http.StatusOK {
		return nil, errors.NewUpstream(fmt.Sprintf("Pinterest API returned status: %d", apiResp.StatusCode))
	}

	body, err := io.ReadAll(apiResp.Body)
	if err != nil {
		return nil, errors.NewUpstream("failed to read Pinterest API response body")
	}

	var parsed struct {
		ResourceResponse struct {
			Data map[string]interface{} `json:"data"`
		} `json:"resource_response"`
	}

	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errors.NewUpstream("failed to parse Pinterest API JSON response")
	}

	data := parsed.ResourceResponse.Data
	if data == nil {
		return nil, errors.NewUpstream("Pinterest API returned empty data object")
	}

	// 3. Extract title, description, and thumbnail
	title := ""
	if t, ok := data["title"].(string); ok && t != "" {
		title = t
	} else if gt, ok := data["grid_title"].(string); ok && gt != "" {
		title = gt
	} else if desc, ok := data["description"].(string); ok && desc != "" {
		title = desc
	}

	thumbnail := ""
	if images, ok := data["images"].(map[string]interface{}); ok {
		if orig, ok := images["orig"].(map[string]interface{}); ok {
			if u, ok := orig["url"].(string); ok {
				thumbnail = u
			}
		} else if size736, ok := images["736x"].(map[string]interface{}); ok {
			if u, ok := size736["url"].(string); ok {
				thumbnail = u
			}
		}
	}

	var downloads []DownloadItem

	// videoQualities defines ordered quality variants to generate direct mp4 URLs.
	// Pinterest stores video segments as CMAF (.cmfv) files which are served as video/mp4.
	type videoQuality struct {
		Suffix  string
		Label   string
	}
	qualities := []videoQuality{
		{"720w", "720p"},
		{"540w", "540p"},
		{"360w", "360p"},
		{"240w", "240p"},
	}

	// reM3U8Sig extracts the video signature from a Pinterest HLS m3u8 URL.
	// e.g. https://v1.pinimg.com/videos/iht/hls/11/09/fe/1109feab967bec9c087b8a1c799ee244.m3u8
	reM3U8Sig := regexp.MustCompile(`/hls/([0-9a-f]{2})/([0-9a-f]{2})/([0-9a-f]{2})/([0-9a-f]{32})\.m3u8`)

	// extractVideoDownloads converts a video_list map into direct MP4 DownloadItems.
	extractVideoDownloads := func(videoList map[string]interface{}) []DownloadItem {
		var items []DownloadItem
		for _, val := range videoList {
			vMap, ok := val.(map[string]interface{})
			if !ok {
				continue
			}
			u, ok := vMap["url"].(string)
			if !ok || u == "" {
				continue
			}
			// Try to extract signature and build direct mp4 URLs
			matches := reM3U8Sig.FindStringSubmatch(u)
			if len(matches) == 5 {
				sig := matches[4]
				p1, p2, p3 := matches[1], matches[2], matches[3]
			baseURL := fmt.Sprintf("https://v1.pinimg.com/videos/iht/hls/%s/%s/%s", p1, p2, p3)
				for _, q := range qualities {
					directURL := fmt.Sprintf("%s/%s_%s.cmfv", baseURL, sig, q.Suffix)
					items = append(items, DownloadItem{
						Quality: q.Label,
						Format:  "mp4",
						URL:     directURL,
					})
				}
				return items // return after first successful signature extraction
			}
			// Fallback: return the m3u8 as-is if no signature found
			items = append(items, DownloadItem{
				Quality: "hls",
				Format:  "m3u8",
				URL:     u,
			})
			return items
		}
		return items
	}

	// 4. Extract video streams (if present at root level)
	videoFound := false
	if videos, ok := data["videos"].(map[string]interface{}); ok && videos != nil {
		if videoList, ok := videos["video_list"].(map[string]interface{}); ok {
			items := extractVideoDownloads(videoList)
			if len(items) > 0 {
				downloads = append(downloads, items...)
				videoFound = true
			}
		}
	}

	// 5. Look for video inside story_pin_data if not found at root level
	if !videoFound {
		if story, ok := data["story_pin_data"].(map[string]interface{}); ok {
			if pages, ok := story["pages"].([]interface{}); ok {
				for _, page := range pages {
					if pMap, ok := page.(map[string]interface{}); ok {
						if blocks, ok := pMap["blocks"].([]interface{}); ok {
							for _, block := range blocks {
								if bMap, ok := block.(map[string]interface{}); ok {
									if videoObj, ok := bMap["video"].(map[string]interface{}); ok {
										if videoList, ok := videoObj["video_list"].(map[string]interface{}); ok {
											items := extractVideoDownloads(videoList)
											if len(items) > 0 {
												downloads = append(downloads, items...)
												videoFound = true
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// 6. If no video found, add the original image as a download option
	if !videoFound && thumbnail != "" {
		downloads = append(downloads, DownloadItem{
			Quality: "original",
			Format:  "image",
			URL:     thumbnail,
		})
	}

	return &Result{
		Platform:  "pinterest",
		Title:     title,
		Thumbnail: thumbnail,
		Downloads: downloads,
	}, nil
}
