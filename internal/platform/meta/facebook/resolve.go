package facebook

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// resolveRedirect follows fb.watch / /share/ short links to their canonical
// permalink. Non-short URLs are returned unchanged. Falls back to canonical /
// og:url meta tags when the HTTP redirect alone doesn't resolve.
func resolveRedirect(ctx context.Context, rawURL, ck string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	path := u.Path
	if !(strings.HasPrefix(u.Host, "fb.watch") || strings.HasPrefix(u.Host, "www.fb.watch") || strings.Contains(path, "/fb.watch/") || strings.Contains(path, "/share/")) {
		return rawURL, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if ck != "" {
		req.Header.Set("Cookie", ck)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	if finalURL != rawURL {
		return finalURL, nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	htmlStr := string(body)

	canonical := regexp.MustCompile(`<link[^>]+rel="canonical"[^>]+href="([^"]+)"`).FindStringSubmatch(htmlStr)
	if len(canonical) > 1 {
		return canonical[1], nil
	}

	ogURL := regexp.MustCompile(`<meta[^>]+property="og:url"[^>]+content="([^"]+)"`).FindStringSubmatch(htmlStr)
	if len(ogURL) > 1 {
		return ogURL[1], nil
	}

	return rawURL, nil
}
