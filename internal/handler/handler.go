package handler

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/pavelc4/astra/internal/cache"
	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/response"
)

// Handlers holds the HTTP layer's injected dependencies. Constructed once at the
// composition root (main) and wired via Register — no package-level handler or
// cache globals.
type Handlers struct {
	cache    *cache.Cache
	sf       singleflight.Group
	freshTTL time.Duration // cached result served directly
	staleTTL time.Duration // served as last-known-good if a refresh fails
	negTTL   time.Duration // failed fetch negatively cached this long
}

// New builds Handlers with a bounded download cache. Tests can construct their
// own instance for an isolated cache. staleTTL stays well under the signed
// lifetime of fbcdn media URLs (a decoded photo `oe` was ~4 days out), so a
// stale response never hands out a dead URL.
func New() *Handlers {
	return &Handlers{
		cache:    cache.New(10_000, 5*time.Minute),
		freshTTL: 15 * time.Minute,
		staleTTL: 1 * time.Hour,
		negTTL:   3 * time.Minute,
	}
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
// argument to reach the cache and the singleflight group.
func download[T any](h *Handlers, fetchFunc func(ctx context.Context, url string) (T, error), successMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawURL := r.URL.Query().Get("url")
		if rawURL == "" {
			response.HandleError(w, errors.NewValidation("url parameter is required"))
			return
		}

		// Key namespaced by operation: the same URL served by different endpoints
		// (e.g. instagram download vs profile) returns different types T, so they
		// must not share a cache entry or a singleflight slot — sharing would
		// hand a follower the wrong type.
		nurl := normalizeURL(rawURL)
		key := successMsg + "|" + nurl

		data, cachedErr, state := h.cache.Get(key)
		if state == cache.Fresh {
			if cachedErr != nil {
				response.HandleError(w, cachedErr)
				return
			}
			if d, ok := data.(T); ok {
				response.OK(w, d, successMsg+" (cached)")
				return
			}
		}

		// singleflight collapses concurrent duplicate fetches for the same key
		// into one upstream call — the key defence against rate-limited scrape
		// targets when a link is requested by many clients at once.
		v, err, _ := h.sf.Do(key, func() (any, error) {
			return fetchFunc(r.Context(), nurl)
		})

		if err != nil {
			// Prefer last-known-good over a transient upstream failure (e.g. a
			// rate-limit 400): if this request saw a stale copy, serve it and
			// keep the stale entry (don't overwrite it with a negative one).
			if state == cache.Stale {
				if d, ok := data.(T); ok {
					response.OK(w, d, successMsg+" (stale)")
					return
				}
			}
			h.cache.SetErr(key, err, h.negTTL)
			response.HandleError(w, err)
			return
		}

		if d, ok := v.(T); ok {
			// Redundant across singleflight followers (same data), but cheap and
			// keeps the write on the request path rather than in the shared fn.
			h.cache.SetOK(key, d, h.freshTTL, h.staleTTL)
			response.OK(w, d, successMsg)
			return
		}
		response.HandleError(w, errors.NewValidation("internal type mismatch"))
	}
}
