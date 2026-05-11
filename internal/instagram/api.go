package instagram

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

func FetchMedia(targetURL string) ([]MediaItem, error) {
	form := url.Values{"url": {targetURL}}

	req, err := http.NewRequest(http.MethodPost, "https://snapsave.app/action.php", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://snapsave.app")
	req.Header.Set("Referer", "https://snapsave.app/")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("SnapSave request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("SnapSave response read failed")
	}

	return parseMedia(body)
}

func parseMedia(data []byte) ([]MediaItem, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, errors.NewUpstream("SnapSave HTML parse failed")
	}

	var items []MediaItem
	doc.Find("tbody tr").Each(func(_ int, s *goquery.Selection) {
		quality := strings.TrimSpace(s.Find("td:nth-child(1)").Text())
		thumbnail, _ := s.Find("img").Attr("src")
		href, ok := s.Find("a").Attr("href")
		if !ok {
			return
		}
		item := MediaItem{Quality: quality, URL: href}
		if thumbnail != "" {
			item.Thumbnail = &thumbnail
		}
		items = append(items, item)
	})

	if len(items) == 0 {
		return nil, errors.NewUpstream("No media found")
	}

	return items, nil
}
