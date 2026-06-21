package facebook

import (
	"net/http"
	"sync"
	"time"

	"github.com/pavelc4/astra/internal/httpclient"
)

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
