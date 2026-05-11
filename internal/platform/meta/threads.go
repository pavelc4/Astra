package meta

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
)

func FetchThreadsData(postURL string) (*ThreadsResult, error) {
	form := url.Values{"q": {postURL}, "t": {"media"}, "lang": {"en"}}

	req, err := http.NewRequest(http.MethodPost, "https://lovethreads.net/api/ajaxSearch", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Origin", "https://lovethreads.net")
	req.Header.Set("Referer", "https://lovethreads.net/en")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	for k, v := range httpclient.DefaultHeaders {
		req.Header[k] = v
	}

	resp, err := httpclient.Client.Do(req)
	if err != nil {
		return nil, errors.NewUpstream(fmt.Sprintf("LoveThreads request failed: %s", err.Error()))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewUpstream("LoveThreads response read failed")
	}

	return parseThreads(body)
}

func parseThreads(data []byte) (*ThreadsResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(data)))
	if err != nil {
		return nil, errors.NewUpstream("LoveThreads HTML parse failed")
	}

	var photos []ThreadsPhoto
	var videos []ThreadsVideo

	doc.Find(".download-box > li").Each(func(_ int, s *goquery.Selection) {
		if s.Find(".icon-dlimage").Length() > 0 {
			var thumbnail *string
			if src, ok := s.Find(".download-items__thumb img").Attr("src"); ok {
				thumbnail = &src
			}

			var variants []ThreadsPhotoVariant
			s.Find(".photo-option option").Each(func(_ int, opt *goquery.Selection) {
				val, ok := opt.Attr("value")
				if !ok {
					return
				}
				label := strings.TrimSpace(opt.Text())
				if !strings.Contains(label, "x") {
					return
				}
				parts := strings.SplitN(label, "x", 2)
				if len(parts) != 2 {
					return
				}
				w, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
				h, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
				variants = append(variants, ThreadsPhotoVariant{
					Resolution: label,
					Width:      w,
					Height:     h,
					URL:        val,
				})
			})

			sort.Slice(variants, func(i, j int) bool {
				return variants[i].Width*variants[i].Height > variants[j].Width*variants[j].Height
			})

			photos = append(photos, ThreadsPhoto{
				Index:     len(photos) + 1,
				Thumbnail: thumbnail,
				Variants:  variants,
			})
		}

		if s.Find(".icon-dlvideo").Length() > 0 {
			var thumbnail *string
			if src, ok := s.Find(".download-items__thumb img").Attr("src"); ok {
				thumbnail = &src
			}

			videoURL, ok := s.Find(`a[title="Download Video"]`).Attr("href")
			if !ok {
				return
			}

			videos = append(videos, ThreadsVideo{
				Index:     len(videos) + 1,
				Thumbnail: thumbnail,
				URL:       videoURL,
				Format:    "mp4",
			})
		}
	})

	return &ThreadsResult{
		Platform:   "threads",
		PhotoCount: len(photos),
		VideoCount: len(videos),
		Photos:     photos,
		Videos:     videos,
	}, nil
}
