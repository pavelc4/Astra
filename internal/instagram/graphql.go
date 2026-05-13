package instagram

import (
	"encoding/json"
	"fmt"
)

func (c *IGClient) FetchProfile(username string) (*UserProfile, error) {
	url := fmt.Sprintf("%s/api/v1/users/web_profile_info/?username=%s", baseURL, username)
	data, err := c.getJSON(url)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			User struct {
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
		} `json:"data"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse profile: %w", err)
	}

	if resp.Data.User.Username == "" {
		return nil, fmt.Errorf("user not found")
	}

	return &UserProfile{
		Username:    resp.Data.User.Username,
		FullName:    resp.Data.User.FullName,
		Avatar:      resp.Data.User.ProfilePicURL,
		AvatarHD:    resp.Data.User.ProfilePicURLHD,
		Biography:   resp.Data.User.Biography,
		Followers:   resp.Data.User.FollowerCount,
		Following:   resp.Data.User.FollowingCount,
		Posts:       resp.Data.User.MediaCount,
		Verified:    resp.Data.User.IsVerified,
		Private:     resp.Data.User.IsPrivate,
		ExternalURL: resp.Data.User.ExternalURL,
	}, nil
}

func (c *IGClient) FetchMedia(longURL string) (*MediaInfo, error) {
	shortcode := extractShortcode(longURL)
	if shortcode == "" {
		return nil, fmt.Errorf("could not extract shortcode from URL")
	}

	url := fmt.Sprintf("%s/p/%s/?__a=1", baseURL, shortcode)
	data, err := c.getJSON(url)
	if err != nil {
		return nil, err
	}

	return parseMediaJSON(data)
}

type graphqlPostData struct {
	GraphQL struct {
		ShortcodeMedia struct {
			Typename    string `json:"__typename"`
			Shortcode   string `json:"shortcode"`
			Caption     string `json:"caption"`
			Owner       struct {
				Username string `json:"username"`
			} `json:"owner"`
			VideoURL      string `json:"video_url"`
			VideoDuration int    `json:"video_duration"`
			DisplayURL    string `json:"display_url"`
			ThumbnailSrc  string `json:"thumbnail_src"`
			IsVideo       bool   `json:"is_video"`
			EdgeMediaToCaption struct {
				Edges []struct {
					Node struct {
						Text string `json:"text"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"edge_media_to_caption"`
			EdgeSidecarToChildren struct {
				Edges []struct {
					Node struct {
						Typename   string `json:"__typename"`
						VideoURL   string `json:"video_url"`
						DisplayURL string `json:"display_url"`
						IsVideo    bool   `json:"is_video"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"edge_sidecar_to_children"`
			ClipsMusicAttributionInfo struct {
				SongName string `json:"song_name"`
				ArtistName string `json:"artist_name"`
				AudioSrc string `json:"audio_src"`
				AudioID   string `json:"audio_id"`
			} `json:"clips_music_attribution_info"`
		} `json:"shortcode_media"`
	} `json:"graphql"`
}

func parseMediaJSON(data []byte) (*MediaInfo, error) {
	var info graphqlPostData
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parse media: %w", err)
	}

	media := info.GraphQL.ShortcodeMedia
	if media.Shortcode == "" {
		return nil, fmt.Errorf("media not found")
	}

	result := &MediaInfo{}

	caption := media.Caption
	if caption == "" && len(media.EdgeMediaToCaption.Edges) > 0 {
		caption = media.EdgeMediaToCaption.Edges[0].Node.Text
	}
	result.Caption = caption
	result.OwnerUser = media.Owner.Username

	if media.ClipsMusicAttributionInfo.AudioSrc != "" {
		result.AudioURL = media.ClipsMusicAttributionInfo.AudioSrc
	}

	if media.Typename == "GraphSidecar" {
		for _, edge := range media.EdgeSidecarToChildren.Edges {
			node := edge.Node
			if node.IsVideo {
				item := MediaItem{Quality: "HD", URL: node.VideoURL}
				if node.DisplayURL != "" {
					item.Thumbnail = &node.DisplayURL
				}
				result.Videos = append(result.Videos, item)
			} else {
				result.Photos = append(result.Photos, MediaItem{URL: node.DisplayURL})
			}
		}
	} else if media.IsVideo {
		item := MediaItem{Quality: "HD", URL: media.VideoURL}
		if media.DisplayURL != "" {
			item.Thumbnail = &media.DisplayURL
		}
		result.Videos = append(result.Videos, item)
	} else {
		result.Photos = append(result.Photos, MediaItem{URL: media.DisplayURL})
	}

	result.Items = append(result.Photos, result.Videos...)

	return result, nil
}
