package handler

import (
	"github.com/pavelc4/astra/internal/platform/capcut"
)

var HandleCapcutDownload = makeDownloadHandler(capcut.FetchData, "CapCut media fetched successfully")
