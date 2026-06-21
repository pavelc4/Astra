package handler

import (
	"github.com/pavelc4/astra/internal/platform/meta"
)

var HandleThreadsDownload = makeDownloadHandler(meta.FetchThreadsData, "Threads media fetched successfully")
