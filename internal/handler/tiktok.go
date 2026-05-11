package handler

import (
	"net/http"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/platform/tiktok"
)

func HandleTikTokDownload(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	data, err := tiktok.FetchData(url)
	if err != nil {
		HandleError(w, err)
		return
	}

	OK(w, data, "TikTok media fetched successfully")
}
