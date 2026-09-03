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
//
// Also returns the page's og:type ("video.*", "article", …) read from this
// light fetch. It matters because the caller can't get it later: the
// authenticated Comet page the crawler fetches carries NO og:type, so a video
// shared as /share/p/ (which resolves to a plain /posts/ URL) would otherwise
// be indistinguishable from a photo post. Empty when unknown / non-short URL.
func resolveRedirect(ctx context.Context, rawURL, ck string) (string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid URL: %w", err)
	}

	path := u.Path
	if !(strings.HasPrefix(u.Host, "fb.watch") || strings.HasPrefix(u.Host, "www.fb.watch") || strings.Contains(path, "/fb.watch/") || strings.Contains(path, "/share/")) {
		return rawURL, "", nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if ck != "" {
		req.Header.Set("Cookie", ck)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read the body first (cheap, 512KB cap) so og:type is available on every
	// path, including the common HTTP-redirect case that used to return early.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	htmlStr := string(body)

	ogType := ""
	if m := ogTypeRe.FindStringSubmatch(htmlStr); len(m) >= 3 {
		if ogType = m[1]; ogType == "" {
			ogType = m[2]
		}
	}

	// Determine the resolved URL: prefer the HTTP redirect, else canonical/og:url.
	finalURL := resp.Request.URL.String()
	if finalURL == rawURL {
		if m := regexp.MustCompile(`<link[^>]+rel="canonical"[^>]+href="([^"]+)"`).FindStringSubmatch(htmlStr); len(m) > 1 {
			finalURL = m[1]
		} else if m := regexp.MustCompile(`<meta[^>]+property="og:url"[^>]+content="([^"]+)"`).FindStringSubmatch(htmlStr); len(m) > 1 {
			finalURL = m[1]
		}
	}

	// The authenticated Comet page (fetched with cookies) omits og:type; a
	// cookie-less probe of the resolved URL reliably carries it. Needed to tell a
	// video shared as /share/p/ (resolves to a plain /posts/ URL) from a real
	// photo post. Skipped for /share/v/ and /share/r/ — the share type already
	// says "video" there, so the caller doesn't need og:type.
	if ogType == "" && !strings.Contains(rawURL, "/share/v/") && !strings.Contains(rawURL, "/share/r/") {
		ogType = probeOgType(ctx, finalURL)
	}
	return finalURL, ogType, nil
}

// probeOgType does a cookie-less GET of rawURL and returns its og:type meta
// ("video.other", "article", …), or "" on any failure. Cookie-less on purpose:
// the authenticated Comet page omits og:type while the public view carries it.
func probeOgType(ctx context.Context, rawURL string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if m := ogTypeRe.FindStringSubmatch(string(body)); len(m) >= 3 {
		if m[1] != "" {
			return m[1]
		}
		return m[2]
	}
	return ""
}
