package facebook

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var (
	// Locates the album/carousel block FB server-renders into the page's
	// RelayPrefetchedStreamCache. count tells us how many photos the post has;
	// each subsequent viewer_image/image uri is one photo.
	albumBlockRe = regexp.MustCompile(`"all_subattachments":\{"count":(\d+)`)
	// viewer_image is the full-resolution variant (carries height/width); image
	// is the smaller thumbnail. Prefer viewer_image, fall back to image.
	albumViewerRe = regexp.MustCompile(`"viewer_image":\{"height":(\d+),"width":(\d+),"uri":"(https:\\?/\\?/scontent[^"]+?)"`)
	albumThumbRe  = regexp.MustCompile(`"image":\{"uri":"(https:\\?/\\?/scontent[^"]+?)"`)
	photoBaseRe   = regexp.MustCompile(`/(\d{6,}_\d+_\d+)_`)
)

// extractAlbumPhotos parses the multi-photo carousel FB embeds in the
// authenticated permalink HTML (StoryAttachmentAlbumStyleRenderer). Returns nil
// if the page has no album block — callers fall back to og:image.
//
// ponytail: scrapes the server-rendered Relay payload, not the GraphQL API —
// FB killed arbitrary q= queries (doc_id-only now), but still SSRs the result
// into the page. If FB stops embedding it, switch to a persisted doc_id call.
func extractAlbumPhotos(htmlStr string) []MediaItem {
	loc := albumBlockRe.FindStringSubmatchIndex(htmlStr)
	if loc == nil {
		return nil
	}
	count := 0
	for _, c := range htmlStr[loc[2]:loc[3]] {
		count = count*10 + int(c-'0')
	}
	if count < 1 {
		return nil
	}

	// Scan only from the album block onward, and cap at count so we don't bleed
	// into a neighbouring post's album further down the page. Dedupe by fbcdn
	// basename (same photo appears as both viewer_image and image).
	tail := htmlStr[loc[1]:]
	seen := make(map[string]bool)
	photos := make([]MediaItem, 0, count)

	for _, m := range albumViewerRe.FindAllStringSubmatch(tail, count*2) {
		clean := cleanJSURL(m[3])
		base := photoBaseName(clean)
		if base == "" || seen[base] {
			continue
		}
		seen[base] = true
		photos = append(photos, MediaItem{URL: clean, Quality: m[2] + "x" + m[1]}) // WxH
		if len(photos) == count {
			return photos
		}
	}

	// Fallback: nodes without a viewer_image — take the thumbnail (no dims).
	for _, m := range albumThumbRe.FindAllStringSubmatch(tail, count*2) {
		clean := cleanJSURL(m[1])
		base := photoBaseName(clean)
		if base == "" || seen[base] {
			continue
		}
		seen[base] = true
		photos = append(photos, MediaItem{URL: clean})
		if len(photos) == count {
			break
		}
	}

	if len(photos) == 0 {
		return nil
	}
	return photos
}

// photoBaseName returns the fbcdn filename (123_456_789_n) so the same photo at
// thumb and full-res resolution dedupes to one entry.
func photoBaseName(u string) string {
	m := photoBaseRe.FindStringSubmatch(u)
	if m == nil {
		return ""
	}
	return m[1]
}

func fetchPhotosViaCrawler(ctx context.Context, targetURL, ck string) (*MediaInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if ck != "" {
		// Full browser headers get FB to server-render the album (bbox) instead
		// of a bare og:image, and avoid the 400 "something went wrong" page.
		setBrowserHeaders(req)
		req.Header.Set("Cookie", ck)
	} else {
		req.Header.Set("User-Agent", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_voiced.php)")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	htmlStr := string(body)
	info := &MediaInfo{}

	// Extract caption
	titleRe := regexp.MustCompile(`<meta[^>]*(?:property|content)="og:title"[^>]*content="([^"]+)"|<meta[^>]*content="([^"]+)"[^>]*(?:property|content)="og:title"`)
	if t := titleRe.FindStringSubmatch(htmlStr); len(t) > 0 {
		val := t[1]
		if val == "" {
			val = t[2]
		}
		info.Caption = html.UnescapeString(val)
	}

	// Multi-photo carousel: prefer the server-rendered album block (all N photos)
	// over og:image (which only ever carries the single cover thumbnail).
	if album := extractAlbumPhotos(htmlStr); len(album) > 0 {
		info.Photos = album
		info.Thumbnail = &info.Photos[0].URL
		// Authenticated album pages carry no og:title — fall back to <title>.
		if info.Caption == "" {
			if t := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`).FindStringSubmatch(htmlStr); len(t) > 1 {
				info.Caption = html.UnescapeString(strings.TrimSpace(t[1]))
			}
		}
		return info, nil
	}

	// Extract photos
	ogImageRe := regexp.MustCompile(`<meta[^>]*(?:property|content)="og:image"[^>]*content="([^"]+)"|<meta[^>]*content="([^"]+)"[^>]*(?:property|content)="og:image"`)
	ogMatches := ogImageRe.FindAllStringSubmatch(htmlStr, -1)

	seen := make(map[string]bool)
	for _, m := range ogMatches {
		val := m[1]
		if val == "" {
			val = m[2]
		}
		clean := strings.ReplaceAll(val, `\/`, `/`)
		clean = strings.ReplaceAll(clean, `&amp;`, `&`)
		if strings.Contains(clean, "static.xx") || strings.Contains(clean, "rsrc.php") {
			continue
		}
		if seen[clean] {
			continue
		}
		seen[clean] = true
		info.Photos = append(info.Photos, MediaItem{URL: clean})
	}

	// Fallback photo extraction for dynamically loaded/Comet page HTML (e.g. private groups/posts)
	if len(info.Photos) == 0 {
		// Post photos typically reside under /t39.xxxx or /v/t39.xxxx and have _n.jpg/_n.png/_n.webp in their filenames
		cometImageRe := regexp.MustCompile(`https?:\\?/\\?/[^\s"'>]+?fbcdn\.net[^\s"'>]+?(?:_n\.[a-zA-Z0-9]+|\/t39\.[^\s"'>]+?\.[a-zA-Z0-9]+)[^\s"'>]*`)
		cometMatches := cometImageRe.FindAllString(htmlStr, -1)

		for _, m := range cometMatches {
			clean := cleanJSURL(m)
			// Skip icons, profile pictures and small thumbnails
			if strings.Contains(clean, "static.xx") || strings.Contains(clean, "rsrc.php") || strings.Contains(clean, "/cp0/") || strings.Contains(clean, "/s40x40/") || strings.Contains(clean, "/s100x100/") {
				continue
			}
			if seen[clean] {
				continue
			}
			seen[clean] = true
			info.Photos = append(info.Photos, MediaItem{URL: clean})
		}
	}

	if len(info.Photos) > 0 {
		info.Thumbnail = &info.Photos[0].URL
	}

	return info, nil
}
