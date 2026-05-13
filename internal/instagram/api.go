package instagram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/httpclient"
)

var cookies string

func SetCookies(c string) {
	cookies = c
}

func fetchWithIG(url string) (*MediaInfo, error) {
	client := NewIGClient(cookies)
	if !client.HasCookies() {
		return nil, fmt.Errorf("no session cookie")
	}
	return client.FetchMedia(url)
}

func fetchWithSnapsave(targetURL string) ([]MediaItem, error) {
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

	return parseSnapsave(body)
}

func parseSnapsave(data []byte) ([]MediaItem, error) {
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

func FetchProfile(username string) (*UserProfile, error) {
	client := NewIGClient(cookies)
	if client.HasCookies() {
		return client.FetchProfile(username)
	}

	return nil, fmt.Errorf("cookies required for profile fetch; set INSTAGRAM_COOKIES")
}

func FetchMedia(targetURL string) (*MediaInfo, error) {
	if cookies != "" {
		if info, err := fetchWithIG(targetURL); err == nil {
			return info, nil
		}
	}

	items, err := fetchWithSnapsave(targetURL)
	if err != nil {
		return nil, err
	}

	return &MediaInfo{Items: items, Photos: items}, nil
}

func FetchProfileFromMedia(longURL string) (*UserProfile, error) {
	client := NewIGClient(cookies)
	if !client.HasCookies() {
		return nil, fmt.Errorf("cookies required")
	}

	shortcode := extractShortcode(longURL)
	if shortcode == "" {
		return nil, fmt.Errorf("could not extract shortcode from URL")
	}

	url := fmt.Sprintf("%s/p/%s/?__a=1", baseURL, shortcode)
	data, err := client.getJSON(url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		GraphQL struct {
			ShortcodeMedia struct {
				Owner struct {
					Username string `json:"username"`
				} `json:"owner"`
			} `json:"shortcode_media"`
		} `json:"graphql"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	if resp.GraphQL.ShortcodeMedia.Owner.Username == "" {
		return nil, fmt.Errorf("could not find owner")
	}

	return client.FetchProfile(resp.GraphQL.ShortcodeMedia.Owner.Username)
}

