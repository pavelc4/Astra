package handler

import (
	"github.com/pavelc4/astra/internal/platform/linkedin"
)

var HandleLinkedinDownload = makeDownloadHandler(linkedin.FetchData, "LinkedIn media fetched successfully")
