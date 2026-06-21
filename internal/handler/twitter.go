package handler

import (
	"github.com/pavelc4/astra/internal/platform/twitter"
)

var HandleTwitterDownload = makeDownloadHandler(twitter.FetchData, "Twitter media fetched successfully")
