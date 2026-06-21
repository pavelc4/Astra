package reddit

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"

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

var reVReddit = regexp.MustCompile(`https://v\.redd\.it/[^\s"<&]+`)
var reImgRedd = regexp.MustCompile(`https://i\.redd\.it/[^\s"<]+`)

func FetchData(ctx context.Context, targetURL string) (*Result, error) {
	// Normalize URL: strip query params and trailing slash for cleaner rapidsave request
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, errors.NewValidation(fmt.Sprintf("invalid URL: %s", err.Error()))
	}
	parsedURL.RawQuery = ""
	cleanURL := strings.TrimRight(parsedURL.String(), "/")

	rapidsaveURL := fmt.Sprintf("https://rapidsave.com/info?url=%s", url.QueryEscape(cleanURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rapidsaveURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Referer", "https://rapidsave.com/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("RapidSave request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("RapidSave response read failed")
	}
	html := string(body)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, errors.NewUpstream("RapidSave HTML parse failed")
	}

	// Extract title from h2.text-center.text-truncate
	title := strings.TrimSpace(doc.Find("h2.text-center.text-truncate").First().Text())

	var downloads []DownloadItem
	var thumbnail string

	doc.Find("a.downloadbutton").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Parse the video_url from the href query string (if present)
		u, err := url.Parse(href)
		videoURL := ""
		if err == nil {
			videoURL = u.Query().Get("video_url")
		}
		if videoURL == "" {
			m := reVReddit.FindString(href)
			if m != "" {
				videoURL = m
			}
		}

		if videoURL != "" {
			format := "mp4"
			quality := "hd"
			if strings.Contains(strings.ToLower(videoURL), "cmaf_720") || strings.Contains(strings.ToLower(videoURL), "720") {
				quality = "720p"
			} else if strings.Contains(strings.ToLower(videoURL), "1080") {
				quality = "1080p"
			}
			downloads = append(downloads, DownloadItem{
				Quality: quality,
				Format:  format,
				URL:     href,
			})
			if thumbnail == "" {
				thumbMatch := doc.Find("img.img-fluid").First()
				if src, ok := thumbMatch.Attr("src"); ok && src != "" {
					thumbnail = src
				}
			}
		} else if reImgRedd.MatchString(href) {
			if thumbnail == "" {
				thumbnail = href
			}
			ext := "image"
			if strings.HasSuffix(href, ".gif") {
				ext = "gif"
			}
			downloads = append(downloads, DownloadItem{
				Quality: "original",
				Format:  ext,
				URL:     href,
			})
		} else if strings.Contains(href, "reddit.com/gallery/") {
			downloads = append(downloads, DownloadItem{
				Quality: "gallery",
				Format:  "link",
				URL:     href,
			})
		}
	})

	if len(downloads) == 0 {
		return nil, errors.NewUpstream("no downloadable media found in Reddit post")
	}

	return &Result{
		Platform:  "reddit",
		Title:     title,
		Thumbnail: thumbnail,
		Downloads: downloads,
	}, nil
}
