package facebook

import (
	"net/http"
	"sync"
	"time"
)

var (
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
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
