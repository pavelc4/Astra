package handler

import (
	"github.com/pavelc4/astra/internal/platform/reddit"
)

var HandleRedditDownload = makeDownloadHandler(reddit.FetchData, "Reddit media fetched successfully")
