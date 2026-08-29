// Package facebook scrapes public and cookie-authenticated Facebook posts,
// videos and multi-photo albums. FetchData is the public entry point; the files
// split the work by concern:
//
//	api.go      — orchestration (FetchMedia/fetchMediaInternal) + shared helpers
//	fetch.go    — FetchData: maps the internal MediaInfo onto media.Media
//	resolve.go  — share/fb.watch short-link resolution
//	photo.go    — og:image + server-rendered album (carousel) extraction
//	video.go    — video ID resolution, progressive/DASH + legacy embed/GraphQL
//	caption.go  — og:title stats-prefix cleanup
//	client.go   — shared HTTP client, cookies, browser headers
//	types.go    — MediaInfo/MediaItem + GraphQL response types
package facebook

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// maxResponseBytes caps how much of an FB page we buffer. Authenticated album/
// watch pages run ~5MB; 16MB leaves headroom while stopping a hostile or broken
// upstream from streaming until we OOM.
const maxResponseBytes = 16 << 20

func FetchMedia(ctx context.Context, targetURL string) (*MediaInfo, error) {
	info, err := fetchMediaInternal(ctx, targetURL)
	if err != nil {
		return nil, err
	}
	if info != nil {
		info.Caption = cleanFacebookCaption(info.Caption)
	}
	return info, nil
}

func fetchMediaInternal(ctx context.Context, targetURL string) (*MediaInfo, error) {
	cookiesMu.RLock()
	ck := cookies
	cookiesMu.RUnlock()

	// 1. Resolve redirect of share/watch URL
	resolvedURL, err := resolveRedirect(ctx, targetURL, ck)
	if err != nil {
		return nil, err
	}

	// Check if we were redirected to a login page
	if strings.Contains(resolvedURL, "/login/") || strings.Contains(resolvedURL, "/login.php") {
		return nil, fmt.Errorf("this post or group is private - login required (please check or refresh your FACEBOOK_COOKIES)")
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
		info, err := fetchVideoViaEmbed(ctx, resolvedURL, ck)
		if err == nil {
			return info, nil
		}

		if isVideo {
			// Fallback to page fetch + extraction/GraphQL if embed fails (only for dedicated video URLs)
			videoID, errID := extractVideoID(ctx, resolvedURL)
			if errID != nil {
				return nil, err
			}

			if ck == "" {
				return nil, fmt.Errorf("FACEBOOK_COOKIES not set — GraphQL API requires authentication")
			}

			pageHTML, dtsg, errPage := fetchPageAndDTSG(ctx, videoID, ck)
			if errPage != nil {
				return nil, errPage
			}

			// Primary modern path: direct MP4s server-rendered into the page.
			if vids := extractProgressiveVideos(pageHTML); len(vids) > 0 {
				info := &MediaInfo{Videos: vids, Caption: extractVideoCaption(pageHTML)}
				if th := regexp.MustCompile(`"preferred_thumbnail":\{"image":\{"uri":"([^"]+)"`).FindStringSubmatch(pageHTML); len(th) > 1 {
					thumb := cleanJSURL(th[1])
					info.Thumbnail = &thumb
					for i := range info.Videos {
						info.Videos[i].Thumbnail = &thumb
					}
				}
				return info, nil
			}

			infoGQL, errGQL := queryGraphQL(ctx, videoID, dtsg, ck)
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

	// 5. For photo/post URLs, fetch as crawler.
	// Prefer cookies first: the multi-photo album is only server-rendered on the
	// authenticated page. Without cookies FB returns a single og:image cover, so
	// a no-cookie-first strategy would stop at 1 photo and never retry (len != 0).
	info, err := fetchPhotosViaCrawler(ctx, resolvedURL, ck)
	if err != nil || len(info.Photos) == 0 {
		if ck != "" {
			// Retry unauthenticated — still yields the og:image cover for public
			// posts if the authenticated fetch got rate-limited (400).
			info, err = fetchPhotosViaCrawler(ctx, resolvedURL, "")
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

func cleanJSURL(u string) string {
	clean := strings.ReplaceAll(u, `\/`, `/`)
	clean = strings.ReplaceAll(clean, "\\u0025", "%")
	clean = strings.ReplaceAll(clean, "\\u0026", "&")
	clean = strings.ReplaceAll(clean, `&amp;`, `&`)
	return clean
}
