package facebook

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var graphQLURL = "https://www.facebook.com/api/graphql/"

var (
	videoIDPatterns = []*regexp.Regexp{
		regexp.MustCompile(`/videos/(?:[^/]+/)?(\d+)/?`),
		regexp.MustCompile(`/reel/(\d+)`),
		regexp.MustCompile(`/watch/?\?.*?v=(\d+)`),
		regexp.MustCompile(`/story\.php\?.*?story_fbid=([^&]+)`),
		regexp.MustCompile(`/permalink\.php\?.*?story_fbid=([^&]+)`),
		regexp.MustCompile(`/posts/([^/?#]+)`),
		regexp.MustCompile(`/photo(?:\.php)?\?.*?fbid=([^&]+)`),
		regexp.MustCompile(`/photos/([^/?#]+)`),
	}
)

func FetchMedia(targetURL string) (*MediaInfo, error) {
	cookiesMu.RLock()
	ck := cookies
	cookiesMu.RUnlock()

	// 1. Resolve redirect of share/watch URL
	resolvedURL, err := resolveRedirect(targetURL)
	if err != nil {
		return nil, err
	}

	// 2. Determine if it is a video URL
	isVideo := false
	isGroup := false
	u, err := url.Parse(resolvedURL)
	if err == nil {
		path := u.Path
		if strings.Contains(path, "/videos/") || strings.Contains(path, "/reel/") || strings.Contains(path, "/watch") || u.Query().Get("v") != "" {
			isVideo = true
		}
		if strings.Contains(path, "/groups/") {
			isGroup = true
		}
	}

	// 3. For video URLs or group URLs (which might contain a video), try to download as video
	if isVideo || isGroup {
		info, err := fetchVideoViaEmbed(resolvedURL, ck)
		if err == nil {
			return info, nil
		}

		if isVideo {
			// Fallback to page fetch + extraction/GraphQL if embed fails (only for dedicated video URLs)
			videoID, errID := extractVideoID(resolvedURL)
			if errID != nil {
				return nil, err
			}

			if ck == "" {
				return nil, fmt.Errorf("FACEBOOK_COOKIES not set — GraphQL API requires authentication")
			}

			pageHTML, dtsg, errPage := fetchPageAndDTSG(videoID, ck)
			if errPage != nil {
				return nil, errPage
			}

			infoGQL, errGQL := queryGraphQL(videoID, dtsg, ck)
			if errGQL == nil {
				return infoGQL, nil
			}

			fallbackInfo, errExt := extractFromPage(pageHTML, videoID)
			if errExt == nil {
				return fallbackInfo, nil
			}

			return nil, err
		}
	}

	// 4. Determine if it is a photo/post URL (or group URL fallback)
	isPhoto := false
	if err == nil {
		path := u.Path
		if strings.Contains(path, "/posts/") || strings.Contains(path, "/photo") || strings.Contains(path, "/permalink") || strings.Contains(resolvedURL, "/story.php") || isGroup {
			isPhoto = true
		}
	}

	if !isPhoto {
		return nil, fmt.Errorf("invalid Facebook URL path: %s", resolvedURL)
	}

	// 5. For photo/post URLs, fetch as crawler
	info, err := fetchPhotosViaCrawler(resolvedURL, "")
	if err != nil || len(info.Photos) == 0 {
		// Retry with cookies if we have them
		if ck != "" {
			info, err = fetchPhotosViaCrawler(resolvedURL, ck)
		}
	}

	if err != nil {
		return nil, err
	}

	if len(info.Photos) == 0 {
		return nil, fmt.Errorf("no media found for Facebook URL: %s", resolvedURL)
	}

	return info, nil
}

func resolveRedirect(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	path := u.Path
	if !(strings.HasPrefix(u.Host, "fb.watch") || strings.HasPrefix(u.Host, "www.fb.watch") || strings.Contains(path, "/fb.watch/") || strings.Contains(path, "/share/")) {
		return rawURL, nil
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

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

func fetchPhotosViaCrawler(targetURL, ck string) (*MediaInfo, error) {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_voiced.php)")
	if ck != "" {
		req.Header.Set("Cookie", ck)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
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

func extractVideoID(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	path := u.Path

	if strings.HasPrefix(u.Host, "fb.watch") || strings.HasPrefix(u.Host, "www.fb.watch") || strings.Contains(path, "/fb.watch/") {
		return resolveShortURL(rawURL, u.Host)
	}

	if strings.Contains(path, "/share/") {
		return resolveShortURL(rawURL, u.Host)
	}

	target := path
	if u.RawQuery != "" {
		target = path + "?" + u.RawQuery
	}

	for _, pat := range videoIDPatterns {
		if m := pat.FindStringSubmatch(target); len(m) > 1 {
			return m[1], nil
		}
	}

	return "", fmt.Errorf("could not extract video ID from URL: %s", rawURL)
}

func resolveShortURL(rawURL, host string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	if finalURL != rawURL {
		return extractVideoID(finalURL)
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	html := string(body)

	canonical := regexp.MustCompile(`<link[^>]+rel="canonical"[^>]+href="([^"]+)"`).FindStringSubmatch(html)
	if len(canonical) > 1 {
		return extractVideoID(canonical[1])
	}

	ogURL := regexp.MustCompile(`<meta[^>]+property="og:url"[^>]+content="([^"]+)"`).FindStringSubmatch(html)
	if len(ogURL) > 1 {
		return extractVideoID(ogURL[1])
	}

	return "", fmt.Errorf("could not resolve short URL: %s", rawURL)
}

func fetchPageAndDTSG(videoID, ck string) (string, string, error) {
	pageURL := fmt.Sprintf("https://www.facebook.com/watch/?v=%s", videoID)

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("create page request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Cookie", ck)

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("page request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("read page: %w", err)
	}

	html := string(body)

	dtsgRe := regexp.MustCompile(`"DTSGInitialData"\s*,\s*\[\]\s*,\s*{\s*"token"\s*:\s*"([^"]+)"`)
	dtsg := ""
	if m := dtsgRe.FindStringSubmatch(html); len(m) > 1 && m[1] != "" {
		dtsg = m[1]
	}

	return html, dtsg, nil
}

func queryGraphQL(videoID, dtsg, ck string) (*MediaInfo, error) {
	query := `query($id: ID!) { node(id: $id) { __typename ... on Video { playable_url playable_url_quality_hd browser_native_hd_url browser_native_sd_url title preferred_thumbnail { image { uri } } playable_duration_in_ms } } }`

	variables := map[string]string{"id": videoID}
	varBytes, _ := json.Marshal(variables)

	form := url.Values{
		"q":         {query},
		"variables": {string(varBytes)},
	}

	if dtsg != "" {
		form.Set("fb_dtsg", dtsg)
	}

	req, err := http.NewRequest(http.MethodPost, graphQLURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create graphql request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Cookie", ck)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read graphql response: %w", err)
	}

	var gql graphQLResponse
	if err := json.Unmarshal(body, &gql); err != nil {
		return nil, fmt.Errorf("parse graphql response: %w", err)
	}

	if len(gql.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", gql.Errors[0].Message)
	}

	if gql.Data == nil || gql.Data.Node == nil {
		return nil, fmt.Errorf("graphql: video not found")
	}

	node := gql.Data.Node
	if node.Typename != "Video" {
		return nil, fmt.Errorf("graphql: expected Video, got %s", node.Typename)
	}

	videoURL := firstNonEmpty(node.PlayableHD, node.PlayableURL, node.BrowserHD, node.BrowserSD)
	if videoURL == "" {
		return nil, fmt.Errorf("graphql: no playable URL found")
	}

	info := &MediaInfo{
		Caption: firstNonEmpty(node.Title, node.Description),
	}

	if node.DurationMs > 0 {
		info.Duration = fmt.Sprintf("%d", node.DurationMs/1000)
	}

	if node.Thumbnail != nil && node.Thumbnail.Image != nil && node.Thumbnail.Image.URI != "" {
		info.Thumbnail = &node.Thumbnail.Image.URI
	}

	info.Videos = []MediaItem{
		{Quality: "hd", URL: videoURL, Thumbnail: info.Thumbnail},
	}

	return info, nil
}

func extractFromPage(html string, videoID string) (*MediaInfo, error) {
	// Attempt to locate playable URLs embedded in RelayPrefetchedData.
	// This is the same approach yt-dlp uses as its primary extraction path.
	relayRe := regexp.MustCompile(`"playable_url(?:_quality_hd)?":"(https?:\\?/\\?/[^"]+)"`)
	matches := relayRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no playable URL found in page data")
	}

	info := &MediaInfo{}

	seen := map[string]bool{}
	for _, m := range matches {
		u := m[1]
		clean := strings.ReplaceAll(u, `\/`, `/`)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		info.Videos = append(info.Videos, MediaItem{URL: clean})
	}

	titleRe := regexp.MustCompile(`"name":"([^"]+)"`)
	if t := titleRe.FindStringSubmatch(html); len(t) > 1 {
		info.Caption = t[1]
	}

	thumbRe := regexp.MustCompile(`"preferred_thumbnail":\{"image":\{"uri":"([^"]+)"\}`)
	if t := thumbRe.FindStringSubmatch(html); len(t) > 1 {
		thumb := strings.ReplaceAll(t[1], `\/`, `/`)
		info.Thumbnail = &thumb
	}

	if len(info.Videos) == 0 {
		return nil, fmt.Errorf("no videos found in page data")
	}

	return info, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return strings.ReplaceAll(v, `\/`, `/`)
		}
	}
	return ""
}

func fetchVideoViaEmbed(targetURL, ck string) (*MediaInfo, error) {
	embedURL := fmt.Sprintf("https://www.facebook.com/plugins/video.php?href=%s", url.QueryEscape(targetURL))
	req, err := http.NewRequest(http.MethodGet, embedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	if ck != "" {
		req.Header.Set("Cookie", ck)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	htmlStr := string(body)

	// Extract sources
	hdRe := regexp.MustCompile(`"hd_src":"([^"]+)"`)
	sdRe := regexp.MustCompile(`"sd_src":"([^"]+)"`)

	var hdURL, sdURL string
	if m := hdRe.FindStringSubmatch(htmlStr); len(m) > 1 {
		hdURL = cleanJSURL(m[1])
	}
	if m := sdRe.FindStringSubmatch(htmlStr); len(m) > 1 {
		sdURL = cleanJSURL(m[1])
	}

	if hdURL == "" && sdURL == "" {
		return nil, fmt.Errorf("no video sources found in embed player")
	}

	info := &MediaInfo{}
	if hdURL != "" {
		info.Videos = append(info.Videos, MediaItem{Quality: "hd", URL: hdURL})
	}
	if sdURL != "" {
		info.Videos = append(info.Videos, MediaItem{Quality: "sd", URL: sdURL})
	}

	// Try fetching title/caption from watch/reel page as crawler
	crawlerInfo, err := fetchPhotosViaCrawler(targetURL, "")
	if err != nil && ck != "" {
		crawlerInfo, err = fetchPhotosViaCrawler(targetURL, ck)
	}
	if err == nil {
		info.Caption = crawlerInfo.Caption
		info.Thumbnail = crawlerInfo.Thumbnail
	} else {
		// Fallback thumbnail/caption extraction from embed page
		titleRe := regexp.MustCompile(`<meta[^>]*(?:property|content)="og:title"[^>]*content="([^"]+)"|<meta[^>]*content="([^"]+)"[^>]*(?:property|content)="og:title"`)
		if t := titleRe.FindStringSubmatch(htmlStr); len(t) > 0 {
			val := t[1]
			if val == "" {
				val = t[2]
			}
			info.Caption = val
		}

		jpgRe := regexp.MustCompile(`"https?:\\?/\\?/[^\s"'>]+?\.jpg[^\s"'>]*"`)
		if m := jpgRe.FindString(htmlStr); m != "" {
			clean := cleanJSURL(strings.Trim(m, `"`))
			info.Thumbnail = &clean
		}
	}

	return info, nil
}

func cleanJSURL(u string) string {
	clean := strings.ReplaceAll(u, `\/`, `/`)
	clean = strings.ReplaceAll(clean, `\u0025`, `%`)
	clean = strings.ReplaceAll(clean, `\u0026`, `&`)
	clean = strings.ReplaceAll(clean, `&amp;`, `&`)
	return clean
}
