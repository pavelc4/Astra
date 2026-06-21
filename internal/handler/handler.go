package handler

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pavelc4/astra/internal/cache"
	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/response"
)

var downloadCache = cache.NewMemoryCache(5 * time.Minute)

const (
	cacheTTL         = 15 * time.Minute
	negativeCacheTTL = 3 * time.Minute
)

type cacheEntry struct {
	data interface{}
	err  error
}

func normalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}

	q := parsed.Query()
	trackingParams := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"igsh", "igshid", "fbclid", "gclid", "share_item", "share_link_id",
		"share_app_id", "share_id", "social_sharing",
	}

	for _, param := range trackingParams {
		q.Del(param)
	}

	parsed.RawQuery = q.Encode()
	parsed.Fragment = ""

	parsed.Host = strings.ToLower(parsed.Host)
	return parsed.String()
}

// makeDownloadHandler creates a standard download handler for a given fetch function.
func makeDownloadHandler[T any](fetchFunc func(ctx context.Context, url string) (T, error), successMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawUrl := r.URL.Query().Get("url")
		if rawUrl == "" {
			response.HandleError(w, errors.NewValidation("url parameter is required"))
			return
		}

		normalizedUrl := normalizeURL(rawUrl)

		if cachedVal, found := downloadCache.Get(normalizedUrl); found {
			if entry, ok := cachedVal.(cacheEntry); ok {
				if entry.err != nil {
					response.HandleError(w, entry.err)
					return
				}
				if typedData, ok := entry.data.(T); ok {
					response.OK(w, typedData, successMsg+" (cached)")
					return
				}
			}
		}

		data, err := fetchFunc(r.Context(), normalizedUrl)
		if err != nil {
			downloadCache.Set(normalizedUrl, cacheEntry{err: err}, negativeCacheTTL)
			response.HandleError(w, err)
			return
		}

		downloadCache.Set(normalizedUrl, cacheEntry{data: data}, cacheTTL)
		response.OK(w, data, successMsg)
	}
}
