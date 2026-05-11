package pinterest

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func FetchData(targetURL string) (*Result, error) {
	fullURL := fmt.Sprintf("https://www.savepin.app/download.php?url=%s&lang=en&type=redirect", url.QueryEscape(targetURL))

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Referer", "https://www.savepin.app/")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("SavePin request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("SavePin response read failed")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, errors.NewUpstream("SavePin HTML parse failed")
	}

	title := strings.TrimSpace(doc.Find("h1").First().Text())
	thumbnail, _ := doc.Find(".image-container img").Attr("src")

	var downloads []DownloadItem
	doc.Find("tbody tr").Each(func(_ int, s *goquery.Selection) {
		quality := strings.TrimSpace(s.Find(".video-quality").Text())
		format := strings.TrimSpace(s.Find("td:nth-child(2)").Text())
		href, ok := s.Find("a").Attr("href")
		if !ok {
			return
		}
		directURL := extractURL(href)
		if quality != "" && format != "" && directURL != "" {
			downloads = append(downloads, DownloadItem{Quality: quality, Format: format, URL: directURL})
		}
	})

	return &Result{
		Platform:  "pinterest",
		Title:     title,
		Thumbnail: thumbnail,
		Downloads: downloads,
	}, nil
}

func extractURL(raw string) string {
	if !strings.Contains(raw, "url=") {
		return raw
	}
	parts := strings.SplitN(raw, "url=", 2)
	if len(parts) < 2 {
		return ""
	}
	decoded, err := url.QueryUnescape(parts[1])
	if err != nil {
		return ""
	}
	return decoded
}
