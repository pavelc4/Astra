package instagram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pavelc4/astra/internal/httpclient"
)

const (
	baseURL  = "https://www.instagram.com"
	apiURL   = "https://i.instagram.com/api/v1"
	mobileUA = "Instagram 347.0.0.30.101 Android (33/13; 540dpi; 1080x2400; samsung; SM-S908E; q5q; qcom; en_US; 679421354)"
	appID    = "936619743392459"
)

var igHTTPClient = &http.Client{
	Timeout:   15 * time.Second,
	Transport: httpclient.Client.Transport,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type IGClient struct {
	http    *http.Client
	cookies string
}

func NewIGClient(cookies string) *IGClient {
	return &IGClient{
		http:    igHTTPClient,
		cookies: cookies,
	}
}

func (c *IGClient) req(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", mobileUA)
	req.Header.Set("Accept", "application/json, text/html, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("X-IG-App-ID", appID)

	if c.cookies != "" {
		req.Header.Set("Cookie", c.cookies)
		parts := strings.Split(c.cookies, ";")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "csrftoken=") {
				req.Header.Set("X-CSRFToken", strings.TrimPrefix(p, "csrftoken="))
			}
		}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

func (c *IGClient) getJSON(ctx context.Context, url string) ([]byte, error) {
	resp, err := c.req(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("instagram returned %d (cookie expired?)", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return data, nil
}

func (c *IGClient) HasCookies() bool {
	return c.cookies != "" && strings.Contains(c.cookies, "sessionid")
}

// extractMediaID parses the media ID from an Instagram page's meta tags
// (<meta property="al:ios:url" content="instagram://media?id=XXX">).
func extractMediaID(ctx context.Context, shortcode string) (string, error) {
	pageURL := fmt.Sprintf("%s/p/%s/", baseURL, shortcode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", fmt.Errorf("create page request: %w", err)
	}
	req.Header.Set("User-Agent", mobileUA)

	resp, err := igHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch page: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read page: %w", err)
	}

	// look for <meta property="al:ios:url" content="instagram://media?id=...">
	marker := `instagram://media?id=`
	idx := strings.Index(string(body), marker)
	if idx == -1 {
		return "", fmt.Errorf("media ID not found in page")
	}

	idStart := idx + len(marker)
	idEnd := strings.IndexByte(string(body)[idStart:], '"')
	if idEnd == -1 {
		return "", fmt.Errorf("malformed media ID")
	}

	return string(body)[idStart : idStart+idEnd], nil
}

// --- Mobile API types ---

type mobileMediaItem struct {
	ID            string `json:"id"`
	Pk            string `json:"pk"`
	Code          string `json:"code"`
	MediaType     int    `json:"media_type"` // 1=photo, 2=video, 8=carousel
	VideoVersions []struct {
		Type   int    `json:"type"`
		URL    string `json:"url"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"video_versions"`
	ImageVersions2 *struct {
		Candidates []struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"candidates"`
	} `json:"image_versions2"`
	CarouselMedia []struct {
		MediaType     int `json:"media_type"`
		VideoVersions []struct {
			Type   int    `json:"type"`
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"video_versions"`
		ImageVersions2 *struct {
			Candidates []struct {
				URL string `json:"url"`
			} `json:"candidates"`
		} `json:"image_versions2"`
	} `json:"carousel_media"`
	Caption *struct {
		Text string `json:"text"`
	} `json:"caption"`
	User *struct {
		Pk       string `json:"pk"`
		Username string `json:"username"`
		FullName string `json:"full_name"`
	} `json:"user"`
	ClipsMetadata *struct {
		MusicInfo *struct {
			AudioSrc string `json:"audio_src"`
			SongName string `json:"song_name"`
		} `json:"music_info"`
	} `json:"clips_metadata"`
}

type mobileInfoResponse struct {
	Items []mobileMediaItem `json:"items"`
}
