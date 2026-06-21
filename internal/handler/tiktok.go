package handler

import (
	"github.com/pavelc4/astra/internal/platform/tiktok"
)

var HandleTikTokDownload = makeDownloadHandler(tiktok.FetchData, "TikTok media fetched successfully")
var HandleTikTokMusic = makeDownloadHandler(tiktok.FetchMusic, "TikTok music fetched successfully")
var HandleTikTokProfile = makeDownloadHandler(tiktok.FetchUser, "TikTok profile fetched successfully")
