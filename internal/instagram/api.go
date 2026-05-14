package instagram

import "fmt"

var cookies string

func SetCookies(c string) {
	cookies = c
}

func FetchProfile(username string) (*UserProfile, error) {
	client := NewIGClient(cookies)
	return client.FetchProfile(username)
}

func FetchMedia(targetURL string) (*MediaInfo, error) {
	client := NewIGClient(cookies)
	return client.FetchMedia(targetURL)
}

func FetchProfileFromMedia(longURL string) (*UserProfile, error) {
	shortcode := extractShortcode(longURL)
	if shortcode == "" {
		return nil, fmt.Errorf("could not extract shortcode from URL")
	}

	client := NewIGClient(cookies)
	info, err := client.FetchMedia(longURL)
	if err != nil {
		return nil, err
	}

	if info.OwnerUser == "" {
		return nil, fmt.Errorf("could not find owner")
	}

	return client.FetchProfile(info.OwnerUser)
}

