package facebook

import (
	"net/http"
	"sync"
	"time"

	"github.com/pavelc4/astra/internal/httpclient"
)

// setBrowserHeaders makes a request look like a real Chrome navigation. FB
// returns a 400 "something went wrong" page (and no server-rendered album) to
// requests that only carry a User-Agent, so the full sec-fetch/sec-ch-ua set
// is required to get the authenticated permalink HTML with the bbox payload.
func setBrowserHeaders(req *http.Request) {
	h := req.Header
	h.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	h.Set("Accept-Language", "en-US,en;q=0.9")
	h.Set("Sec-Ch-Ua", `"Chromium";v="120", "Not:A-Brand";v="99"`)
	h.Set("Sec-Ch-Ua-Mobile", "?0")
	h.Set("Sec-Ch-Ua-Platform", `"Linux"`)
	h.Set("Sec-Fetch-Dest", "document")
	h.Set("Sec-Fetch-Mode", "navigate")
	h.Set("Sec-Fetch-Site", "none")
	h.Set("Sec-Fetch-User", "?1")
	h.Set("Upgrade-Insecure-Requests", "1")
}

var (
	httpClient = &http.Client{
		Timeout:   30 * time.Second,
		Transport: httpclient.Client.Transport,
	}

	cookies   string
	cookiesMu sync.RWMutex
)

func SetCookies(c string) {
	cookiesMu.Lock()
	defer cookiesMu.Unlock()
	cookies = c
}

func HasCookies() bool {
	cookiesMu.RLock()
	defer cookiesMu.RUnlock()
	return cookies != ""
}
