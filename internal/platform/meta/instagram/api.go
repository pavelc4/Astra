package instagram

import (
	"context"
	"fmt"
	"sync"
)

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

func FetchProfile(ctx context.Context, username string) (*UserProfile, error) {
	client := NewIGClient(GetCookies())
	return client.FetchProfile(ctx, username)
}

func FetchMedia(ctx context.Context, targetURL string) (*MediaInfo, error) {
	client := NewIGClient(GetCookies())
	return client.FetchMedia(ctx, targetURL)
}

func FetchStories(ctx context.Context, username string) (*StoriesResult, error) {
	client := NewIGClient(GetCookies())
	return client.FetchStories(ctx, username)
}

func FetchProfileFromMedia(ctx context.Context, longURL string) (*UserProfile, error) {
	shortcode := extractShortcode(longURL)
	if shortcode == "" {
		return nil, fmt.Errorf("could not extract shortcode from URL")
	}

	client := NewIGClient(GetCookies())
	info, err := client.FetchMedia(ctx, longURL)
	if err != nil {
		return nil, err
	}

	if info.OwnerUser == "" {
		return nil, fmt.Errorf("could not find owner")
	}

	return client.FetchProfile(ctx, info.OwnerUser)
}
