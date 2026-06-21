package youtube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/types"
)

type YtFormat struct {
	URL        string `json:"url"`
	Ext        string `json:"ext"`
	FormatNote string `json:"format_note"`
	Resolution string `json:"resolution"`
	VCodec     string `json:"vcodec"`
	ACodec     string `json:"acodec"`
	FileSize   int64  `json:"filesize"`
}

type YtDump struct {
	Title     string     `json:"title"`
	Thumbnail string     `json:"thumbnail"`
	Formats   []YtFormat `json:"formats"`
}

type Result struct {
	Platform  string               `json:"platform"`
	Title     string               `json:"title"`
	Thumbnail string               `json:"thumbnail,omitempty"`
	Downloads []types.DownloadItem `json:"downloads"`
}

func FetchData(ctx context.Context, targetURL string) (*Result, error) {
	// 1. Prepare arguments for yt-dlp
	args := []string{"--dump-json", targetURL}

	// 2. Check for cookie file
	cookieFile := os.Getenv("YOUTUBE_COOKIES_FILE")
	if cookieFile == "" {
		// Fallbacks
		if _, err := os.Stat("yt-cookies.txt"); err == nil {
			cookieFile = "yt-cookies.txt"
		} else if _, err := os.Stat("youtube-cookies.txt"); err == nil {
			cookieFile = "youtube-cookies.txt"
		}
	}

	if cookieFile != "" {
		args = append([]string{"--cookies", cookieFile}, args...)
	}

	// 3. Execute command with context (hybrid check: env path -> local ./yt-dlp binary -> system yt-dlp)
	ytDlpPath := os.Getenv("YT_DLP_PATH")
	if ytDlpPath == "" {
		if _, err := os.Stat("./yt-dlp"); err == nil {
			ytDlpPath = "./yt-dlp"
		} else {
			ytDlpPath = "yt-dlp"
		}
	}

	cmd := exec.CommandContext(ctx, ytDlpPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if strings.Contains(stderrStr, "executable file not found") || strings.Contains(err.Error(), "executable file not found") {
			return nil, errors.NewUpstream("yt-dlp is not installed on the server. Please install yt-dlp and ffmpeg.")
		}
		return nil, errors.NewUpstream(fmt.Sprintf("yt-dlp failed: %s %s", err.Error(), stderrStr))
	}

	// 4. Parse JSON output
	var dump YtDump
	if err := json.Unmarshal(stdout.Bytes(), &dump); err != nil {
		return nil, fmt.Errorf("failed to parse yt-dlp JSON: %w", err)
	}

	var downloads []types.DownloadItem

	// 5. Categorize and map formats
	for _, f := range dump.Formats {
		if f.URL == "" {
			continue
		}

		isAudio := f.VCodec == "none"
		isVideoOnly := f.ACodec == "none"
		isCombined := f.VCodec != "none" && f.ACodec != "none" && f.VCodec != "" && f.ACodec != ""

		if isCombined {
			quality := f.FormatNote
			if quality == "" {
				quality = f.Resolution
			}
			label := fmt.Sprintf("Video %s (%s)", quality, f.Ext)
			downloads = append(downloads, types.DownloadItem{
				Label:   label,
				URL:     f.URL,
				Type:    types.MediaVideo,
				Quality: quality,
			})
		} else if isAudio {
			label := fmt.Sprintf("Audio (%s)", f.Ext)
			downloads = append(downloads, types.DownloadItem{
				Label:   label,
				URL:     f.URL,
				Type:    types.MediaAudio,
				Quality: f.FormatNote,
			})
		} else if isVideoOnly {
			quality := f.FormatNote
			if quality == "" {
				quality = f.Resolution
			}
			label := fmt.Sprintf("Video %s (no audio) (%s)", quality, f.Ext)
			downloads = append(downloads, types.DownloadItem{
				Label:   label,
				URL:     f.URL,
				Type:    types.MediaVideo,
				Quality: quality,
			})
		}
	}

	if len(downloads) == 0 {
		return nil, errors.NewUpstream("no downloadable formats found for this YouTube URL")
	}

	return &Result{
		Platform:  "youtube",
		Title:     dump.Title,
		Thumbnail: dump.Thumbnail,
		Downloads: downloads,
	}, nil
}
