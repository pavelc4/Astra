package handler

import (
	"github.com/pavelc4/astra/internal/platform/meta"
)

var HandleFacebookDownload = makeDownloadHandler(meta.FetchFacebookData, "Facebook media fetched successfully")
