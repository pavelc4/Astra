package tiktok

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
	"github.com/pavelc4/astra/internal/types"
)

type Result struct {
	Platform  string               `json:"platform"`
	Title     *string              `json:"title"`
	Thumbnail *string              `json:"thumbnail"`
	Downloads []types.DownloadItem `json:"downloads"`
}

func FetchData(videoURL string) (*Result, error) {
	form := url.Values{"id": {videoURL}, "locale": {"en"}, "tt": {"dHl6Ylg4"}}

	req, err := http.NewRequest(http.MethodPost, "https://ssstik.io/abc?url=dl", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", "https://ssstik.io/en-1")
	req.Header.Set("Hx-Request", "true")
	req.Header.Set("Hx-Target", "target")
	req.Header.Set("Hx-Trigger", "_gcaptcha_pt")

	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("SSSTik request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("SSSTik response read failed")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, errors.NewUpstream("SSSTik HTML parse failed")
	}

	return parseResult(doc), nil
}

func parseResult(doc *goquery.Document) *Result {
	var title *string
	if t := strings.TrimSpace(doc.Find("#avatar_and_text h2").Text()); t != "" {
		title = &t
	}
	if title == nil {
		if t := strings.TrimSpace(doc.Find("#avatarAndTextUsual h2").Text()); t != "" {
			title = &t
		}
	}

	var thumbnail *string
	if src, ok := doc.Find(".result_author").Attr("src"); ok {
		thumbnail = &src
	}

	var downloads []types.DownloadItem
	doc.Find("a.download_link:not(.slide)").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok || href == "" || href == "#" {
			return
		}
		label := strings.TrimSpace(s.Text())
		downloads = append(downloads, types.DownloadItem{Label: label, URL: href, Type: types.MediaVideo})
	})
	doc.Find("a.download_link.slide").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok || href == "" || href == "#" {
			return
		}
		downloads = append(downloads, types.DownloadItem{URL: href, Type: types.MediaSlide})
	})

	return &Result{
		Platform:  "tiktok",
		Title:     title,
		Thumbnail: thumbnail,
		Downloads: downloads,
	}
}
