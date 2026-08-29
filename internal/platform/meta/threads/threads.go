package threads

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
)

func FetchData(ctx context.Context, postURL string) (*ThreadsResult, error) {
	var caption string
	var fallbackImage string

	// 1. Fetch the Threads post page directly with a crawler User-Agent to extract metadata (caption & primary image)
	reqMeta, err := http.NewRequestWithContext(ctx, http.MethodGet, postURL, nil)
	if err == nil {
		reqMeta.Header.Set("User-Agent", "facebookexternalhit/1.1")
		respMeta, errMeta := httpclient.Client.Do(reqMeta)
		if errMeta == nil && respMeta.StatusCode == 200 {
			defer respMeta.Body.Close()
			metaBytes, _ := io.ReadAll(respMeta.Body)
			metaStr := string(metaBytes)

			// Extract caption
			reDesc := regexp.MustCompile(`<meta\s+property="og:description"\s+content="([^"]*)"`)
			if m := reDesc.FindStringSubmatch(metaStr); len(m) > 1 {
				caption = html.UnescapeString(m[1])
			} else {
				reDescAlt := regexp.MustCompile(`<meta\s+name="description"\s+content="([^"]*)"`)
				if mAlt := reDescAlt.FindStringSubmatch(metaStr); len(mAlt) > 1 {
					caption = html.UnescapeString(mAlt[1])
				}
			}

			// Extract fallback primary image
			reImg := regexp.MustCompile(`<meta\s+property="og:image"\s+content="([^"]*)"`)
			if mImg := reImg.FindStringSubmatch(metaStr); len(mImg) > 1 {
				fallbackImage = html.UnescapeString(mImg[1])
			}
		}
	}

	var photos []ThreadsPhoto
	var videos []ThreadsVideo

	// 2. Fetch SSSThreads homepage to get the current AJAX URL and Nonce
	reqHome, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://sssthreads.cc/", nil)
	if err == nil {
		reqHome.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		respHome, errHome := httpclient.Client.Do(reqHome)
		if errHome == nil && respHome.StatusCode == 200 {
			defer respHome.Body.Close()
			homeBytes, _ := io.ReadAll(respHome.Body)
			homeStr := string(homeBytes)

			reConfig := regexp.MustCompile(`tvdFrontendData\s*=\s*({[^;]+});?`)
			if mConfig := reConfig.FindStringSubmatch(homeStr); len(mConfig) > 1 {
				var tvdConfig struct {
					AjaxURL string `json:"ajax_url"`
					Nonce   string `json:"nonce"`
				}
				if json.Unmarshal([]byte(mConfig[1]), &tvdConfig) == nil {
					// 3. Post to the AJAX endpoint
					form := url.Values{}
					form.Set("action", "tvd_request_download")
					form.Set("nonce", tvdConfig.Nonce)
					form.Set("url", postURL)

					reqAjax, errAjax := http.NewRequestWithContext(ctx, http.MethodPost, tvdConfig.AjaxURL, bytes.NewBufferString(form.Encode()))
					if errAjax == nil {
						reqAjax.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
						reqAjax.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
						reqAjax.Header.Set("Origin", "https://sssthreads.cc")
						reqAjax.Header.Set("Referer", "https://sssthreads.cc/")
						reqAjax.Header.Set("X-Requested-With", "XMLHttpRequest")

						respAjax, errAjaxCall := httpclient.Client.Do(reqAjax)
						if errAjaxCall == nil && respAjax.StatusCode == 200 {
							defer respAjax.Body.Close()
							ajaxBytes, _ := io.ReadAll(respAjax.Body)
							var ajaxRes struct {
								Success bool `json:"success"`
								Data    struct {
									HTML string `json:"html"`
								} `json:"data"`
							}
							if json.Unmarshal(ajaxBytes, &ajaxRes) == nil && ajaxRes.Success {
								// Parse SSSThreads HTML snippet
								doc, errDoc := goquery.NewDocumentFromReader(strings.NewReader(ajaxRes.Data.HTML))
								if errDoc == nil {
									// Extract author caption if not set
									if caption == "" {
										if titleText := doc.Find(".tvd-author-title").Text(); titleText != "" {
											caption = strings.TrimSpace(titleText)
										}
									}

									// Extract media items
									doc.Find(".tvd-result-item").Each(func(_ int, s *goquery.Selection) {
										btn := s.Find("a.tvd-download-button")
										downloadURL, ok := btn.Attr("href")
										if !ok {
											return
										}

										// Extract encoded direct media URL from query parameters
										parsedURL, errParse := url.Parse(downloadURL)
										if errParse != nil {
											return
										}
										encodedURL := parsedURL.Query().Get("u")
										if encodedURL == "" {
											return
										}

										directURL, errDecode := decodeBase64URL(encodedURL)
										if errDecode != nil {
											return
										}

										labelType, _ := s.Find(".tvd-thumbnail-wrapper").Attr("data-label-type")
										isVid := labelType == "video" || strings.Contains(directURL, ".mp4") || strings.Contains(directURL, "mime=video")

										if isVid {
											videos = append(videos, ThreadsVideo{
												Index:  len(videos) + 1,
												URL:    directURL,
												Format: "mp4",
											})
										} else {
											photos = append(photos, ThreadsPhoto{
												Index: len(photos) + 1,
												Variants: []ThreadsPhotoVariant{
													{
														Resolution: "highest",
														URL:        directURL,
													},
												},
											})
										}
									})
								}
							}
						}
					}
				}
			}
		}
	}

	// 4. Fallback logic: if no photos or videos were found via SSSThreads, but we have a fallback primary image
	if len(photos) == 0 && len(videos) == 0 && fallbackImage != "" {
		isProfilePic := strings.Contains(fallbackImage, "profile_pic") ||
			strings.Contains(fallbackImage, "-19/") ||
			strings.Contains(fallbackImage, "_19_n.") ||
			strings.Contains(fallbackImage, "150x150") ||
			strings.Contains(fallbackImage, "/t51.2885-19/") ||
			strings.Contains(fallbackImage, "/t51.82787-19/")

		if !isProfilePic {
			photos = append(photos, ThreadsPhoto{
				Index: 1,
				Variants: []ThreadsPhotoVariant{
					{
						Resolution: "highest",
						URL:        fallbackImage,
					},
				},
			})
		}
	}

	// If everything is completely empty (no caption, no photos, no videos), return error
	if caption == "" && len(photos) == 0 && len(videos) == 0 {
		return nil, errors.NewUpstream("failed to extract any Threads post data")
	}

	return &ThreadsResult{
		Platform:   "threads",
		Caption:    caption,
		PhotoCount: len(photos),
		VideoCount: len(videos),
		Photos:     photos,
		Videos:     videos,
	}, nil
}

func decodeBase64URL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw)%4 != 0 {
		raw += strings.Repeat("=", 4-(len(raw)%4))
	}
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		data, err = base64.URLEncoding.DecodeString(raw)
		if err != nil {
			return "", err
		}
	}
	return string(data), nil
}
