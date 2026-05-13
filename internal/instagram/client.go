package instagram

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pavelc4/astra/internal/httpclient"
)

const (
	baseURL = "https://www.instagram.com"
	apiURL  = "https://i.instagram.com/api/v1"
)

type IGClient struct {
	http    *http.Client
	cookies string
}

func NewIGClient(cookies string) *IGClient {
	return &IGClient{
		http:    httpclient.Client,
		cookies: cookies,
	}
}

func (c *IGClient) req(method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36")
	req.Header.Set("Accept", "application/json, text/html, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.instagram.com/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	if c.cookies != "" {
		req.Header.Set("Cookie", c.cookies)
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

func (c *IGClient) getJSON(url string) ([]byte, error) {
	resp, err := c.req(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return data, nil
}

func (c *IGClient) HasCookies() bool {
	return c.cookies != "" && strings.Contains(c.cookies, "sessionid")
}
