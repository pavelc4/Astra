package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
	"github.com/pavelc4/astra/internal/types"
)

type Result struct {
	Platform  string               `json:"platform"`
	Type      string               `json:"type"`
	TweetID   *string              `json:"tweetId"`
	Title     *string              `json:"title"`
	Duration  *string              `json:"duration"`
	Thumbnail *string              `json:"thumbnail"`
	Downloads []types.DownloadItem `json:"downloads"`
}

type AjaxResponse struct {
	Status string `json:"status"`
	Data   string `json:"data"`
}

func FetchData(ctx context.Context, tweetURL string) (*Result, error) {
	form := url.Values{"q": {tweetURL}, "lang": {"en"}, "cftoken": {""}}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://savetwitter.net/api/ajaxSearch", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", "https://savetwitter.net")
	req.Header.Set("Referer", "https://savetwitter.net/en4")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("SaveTwitter request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("SaveTwitter response read failed")
	}

	var ajaxResp AjaxResponse
	if err := json.Unmarshal(body, &ajaxResp); err != nil {
		return nil, errors.NewUpstream("failed to parse SaveTwitter JSON response")
	}

	return parseResult([]byte(ajaxResp.Data)), nil
}

var qualityRe = regexp.MustCompile(`\((\d+p)\)`)

func parseResult(data []byte) *Result {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return &Result{Platform: "twitter", Type: "photo"}
	}

	var tweetID *string
	if v, ok := doc.Find("#TwitterId").Attr("value"); ok {
		tweetID = &v
	}

	var title *string
	if t := strings.TrimSpace(doc.Find(".tw-middle h3").First().Text()); t != "" {
		title = &t
	}

	var duration *string
	if d := strings.TrimSpace(doc.Find(".tw-middle p").First().Text()); d != "" {
		duration = &d
	}

	var thumbnail *string
	if src, ok := doc.Find(".thumbnail img").Attr("src"); ok {
		thumbnail = &src
	}
	if thumbnail == nil {
		if src, ok := doc.Find(".download-items__thumb img").Attr("src"); ok {
			thumbnail = &src
		}
	}

	var downloads []types.DownloadItem
	doc.Find(".tw-button-dl").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok || !strings.Contains(href, "dl.snapcdn.app") {
			return
		}
		text := strings.TrimSpace(s.Text())

		if strings.Contains(text, "MP4") {
			quality := ""
			if m := qualityRe.FindStringSubmatch(text); len(m) > 1 {
				quality = m[1]
			}
			downloads = append(downloads, types.DownloadItem{
				Label:   fmt.Sprintf("Video %s", quality),
				Quality: quality,
				URL:     href,
				Type:    types.MediaVideo,
			})
		}
		if strings.Contains(text, "图片") || strings.Contains(strings.ToLower(text), "photo") || strings.Contains(strings.ToLower(text), "image") {
			downloads = append(downloads, types.DownloadItem{
				Label:   "Photo",
				URL:     href,
				Type:    types.MediaImage,
				Quality: "original",
			})
		}
	})

	doc.Find(".photo-list img").Each(func(_ int, s *goquery.Selection) {
		src, ok := s.Attr("src")
		if ok {
			downloads = append(downloads, types.DownloadItem{
				Label:   "Photo",
				URL:     src,
				Type:    types.MediaImage,
				Quality: "original",
			})
		}
	})

	sort.SliceStable(downloads, func(i, j int) bool {
		if downloads[i].Type != types.MediaVideo || downloads[j].Type != types.MediaVideo {
			return false
		}
		qi, _ := strconv.Atoi(strings.TrimSuffix(downloads[i].Quality, "p"))
		qj, _ := strconv.Atoi(strings.TrimSuffix(downloads[j].Quality, "p"))
		return qi > qj
	})

	resultType := "photo"
	for _, d := range downloads {
		if d.Type == types.MediaVideo {
			resultType = "video"
			break
		}
	}

	return &Result{
		Platform:  "twitter",
		Type:      resultType,
		TweetID:   tweetID,
		Title:     title,
		Duration:  duration,
		Thumbnail: thumbnail,
		Downloads: downloads,
	}
}
