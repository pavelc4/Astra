package handler

import (
	"github.com/pavelc4/astra/internal/platform/youtube"
)

var HandleYoutubeDownload = makeDownloadHandler(youtube.FetchData, "YouTube media fetched successfully")
