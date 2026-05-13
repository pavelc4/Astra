package instagram

import "strings"

type MediaItem struct {
	Quality   string  `json:"quality"`
	Thumbnail *string `json:"thumbnail,omitempty"`
	URL       string  `json:"url"`
}

type UserProfile struct {
	Username      string `json:"username"`
	FullName      string `json:"full_name"`
	Avatar        string `json:"avatar"`
	AvatarHD      string `json:"avatar_hd"`
	Biography     string `json:"biography"`
	Followers     int    `json:"followers"`
	Following     int    `json:"following"`
	Posts         int    `json:"posts"`
	Verified      bool   `json:"verified"`
	Private       bool   `json:"private"`
	ExternalURL   string `json:"external_url,omitempty"`
}

type MediaInfo struct {
	Items       []MediaItem        `json:"items"`
	Caption     string             `json:"caption,omitempty"`
	OwnerAvatar string             `json:"owner_avatar,omitempty"`
	OwnerUser   string             `json:"owner_username,omitempty"`
	AudioURL    string             `json:"audio_url,omitempty"`
	Photos      []MediaItem        `json:"photos,omitempty"`
	Videos      []MediaItem        `json:"videos,omitempty"`
}

func extractShortcode(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)

	if idx := strings.LastIndex(rawURL, "/p/"); idx != -1 {
		s := rawURL[idx+3:]
		if e := strings.Index(s, "/"); e != -1 {
			s = s[:e]
		}
		if e := strings.Index(s, "?"); e != -1 {
			s = s[:e]
		}
		return s
	}
	if idx := strings.LastIndex(rawURL, "/reel/"); idx != -1 {
		s := rawURL[idx+6:]
		if e := strings.Index(s, "/"); e != -1 {
			s = s[:e]
		}
		if e := strings.Index(s, "?"); e != -1 {
			s = s[:e]
		}
		return s
	}

	return ""
}


