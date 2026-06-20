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

	// 4. Extract video streams (if present)
	videoFound := false
	if videos, ok := data["videos"].(map[string]interface{}); ok && videos != nil {
		if videoList, ok := videos["video_list"].(map[string]interface{}); ok {
			for key, val := range videoList {
				if vMap, ok := val.(map[string]interface{}); ok {
					if u, ok := vMap["url"].(string); ok && u != "" {
						quality := key
						format := "mp4"
						if regexp.MustCompile(`(?i)\.m3u8`).MatchString(u) {
							format = "m3u8"
						}
						downloads = append(downloads, DownloadItem{
							Quality: quality,
							Format:  format,
							URL:     u,
						})
						videoFound = true
					}
				}
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
											for key, val := range videoList {
												if vMap, ok := val.(map[string]interface{}); ok {
													if u, ok := vMap["url"].(string); ok && u != "" {
														quality := key
														format := "mp4"
														if regexp.MustCompile(`(?i)\.m3u8`).MatchString(u) {
															format = "m3u8"
														}
														downloads = append(downloads, DownloadItem{
															Quality: quality,
															Format:  format,
															URL:     u,
														})
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
