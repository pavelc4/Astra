package handler

import (
	"github.com/pavelc4/astra/internal/platform/pinterest"
)

var HandlePinterestDownload = makeDownloadHandler(pinterest.FetchData, "Pinterest media fetched successfully")
