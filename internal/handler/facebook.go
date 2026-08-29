package handler

import (
	"github.com/pavelc4/astra/internal/platform/meta/facebook"
)

var HandleFacebookDownload = makeDownloadHandler(facebook.FetchData, "Facebook media fetched successfully")
