package instagram

import (
	"encoding/json"
	"fmt"
)

func (c *IGClient) FetchProfile(username string) (*UserProfile, error) {
	url := fmt.Sprintf("%s/users/%s/usernameinfo/", apiURL, username)
	data, err := c.getJSON(url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		User struct {
			Pk              string `json:"pk"`
			Username        string `json:"username"`
			FullName        string `json:"full_name"`
			ProfilePicURL   string `json:"profile_pic_url"`
			ProfilePicURLHD string `json:"profile_pic_url_hd"`
			Biography       string `json:"biography"`
			FollowerCount   int    `json:"follower_count"`
			FollowingCount  int    `json:"following_count"`
			MediaCount      int    `json:"media_count"`
			IsVerified      bool   `json:"is_verified"`
			IsPrivate       bool   `json:"is_private"`
			ExternalURL     string `json:"external_url"`
		} `json:"user"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}

	if resp.User.Username == "" {
		return nil, fmt.Errorf("user not found")
	}

	return &UserProfile{
		Username:    resp.User.Username,
		FullName:    resp.User.FullName,
		Avatar:      resp.User.ProfilePicURL,
		AvatarHD:    resp.User.ProfilePicURLHD,
		Biography:   resp.User.Biography,
		Followers:   resp.User.FollowerCount,
		Following:   resp.User.FollowingCount,
		Posts:       resp.User.MediaCount,
		Verified:    resp.User.IsVerified,
		Private:     resp.User.IsPrivate,
		ExternalURL: resp.User.ExternalURL,
	}, nil
}

func (c *IGClient) FetchMedia(longURL string) (*MediaInfo, error) {
	shortcode := extractShortcode(longURL)
	if shortcode == "" {
		return nil, fmt.Errorf("could not extract shortcode from URL")
	}

	mediaID, err := extractMediaID(shortcode)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/media/%s/info/", apiURL, mediaID)
	data, err := c.getJSON(url)
	if err != nil {
		return nil, err
	}

	var resp mobileInfoResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse media: %w", err)
	}

	if len(resp.Items) == 0 {
		return nil, fmt.Errorf("media not found")
	}

	return convertMediaItem(&resp.Items[0]), nil
}

func convertMediaItem(item *mobileMediaItem) *MediaInfo {
	result := &MediaInfo{}

	if item.Caption != nil {
		result.Caption = item.Caption.Text
	}
	if item.User != nil {
		result.OwnerUser = item.User.Username
	}

	if item.MediaType == 2 {
		// single video
		for _, v := range item.VideoVersions {
			result.Videos = append(result.Videos, MediaItem{
				Quality: fmt.Sprintf("%dx%d", v.Width, v.Height),
				URL:     v.URL,
			})
		}
	} else if item.MediaType == 8 {
		// carousel
		for _, cm := range item.CarouselMedia {
			if cm.MediaType == 2 {
				for _, v := range cm.VideoVersions {
					result.Videos = append(result.Videos, MediaItem{URL: v.URL})
				}
			} else {
				if cm.ImageVersions2 != nil && len(cm.ImageVersions2.Candidates) > 0 {
					result.Photos = append(result.Photos, MediaItem{URL: cm.ImageVersions2.Candidates[0].URL})
				}
			}
		}
	} else {
		// single photo
		if item.ImageVersions2 != nil && len(item.ImageVersions2.Candidates) > 0 {
			result.Photos = append(result.Photos, MediaItem{URL: item.ImageVersions2.Candidates[0].URL})
		}
	}

	// audio from clips metadata
	if item.ClipsMetadata != nil && item.ClipsMetadata.MusicInfo != nil {
		result.AudioURL = item.ClipsMetadata.MusicInfo.AudioSrc
	}

	result.Items = append(result.Photos, result.Videos...)

	return result
}
