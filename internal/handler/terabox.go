package handler

import (
	"github.com/pavelc4/astra/internal/platform/terabox"
)

var HandleTeraboxDownload = makeDownloadHandler(terabox.FetchData, "TeraBox media fetched successfully")
