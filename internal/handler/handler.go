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

// Handlers holds the HTTP layer's injected dependencies. Constructed once at the
// composition root (main) and wired via Register — no package-level handler or
// cache globals.
type Handlers struct {
	cache            *cache.MemoryCache
	cacheTTL         time.Duration
	negativeCacheTTL time.Duration
}

// New builds Handlers with a fresh in-memory download cache. Tests can construct
// their own instance to get an isolated cache.
func New() *Handlers {
	return &Handlers{
		cache:            cache.NewMemoryCache(5 * time.Minute),
		cacheTTL:         15 * time.Minute,
		negativeCacheTTL: 3 * time.Minute,
	}
}

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

// download builds a caching download handler for a fetch function. Go doesn't
// allow generic methods, so the injected *Handlers is passed as the first
// argument to reach the request cache.
func download[T any](h *Handlers, fetchFunc func(ctx context.Context, url string) (T, error), successMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawUrl := r.URL.Query().Get("url")
		if rawUrl == "" {
			response.HandleError(w, errors.NewValidation("url parameter is required"))
			return
		}

		normalizedUrl := normalizeURL(rawUrl)

		if cachedVal, found := h.cache.Get(normalizedUrl); found {
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
			h.cache.Set(normalizedUrl, cacheEntry{err: err}, h.negativeCacheTTL)
			response.HandleError(w, err)
			return
		}

		h.cache.Set(normalizedUrl, cacheEntry{data: data}, h.cacheTTL)
		response.OK(w, data, successMsg)
	}
}
