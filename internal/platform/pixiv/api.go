package pixiv

import "sync"

var (
	cookies   string
	cookiesMu sync.RWMutex
)

func SetCookies(c string) {
	cookiesMu.Lock()
	defer cookiesMu.Unlock()
	cookies = c
}

func GetCookies() string {
	cookiesMu.RLock()
	defer cookiesMu.RUnlock()
	return cookies
}
