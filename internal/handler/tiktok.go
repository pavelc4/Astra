package handler

import (
	"net/http"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/platform/tiktok"
	"github.com/pavelc4/astra/internal/response"
)

func HandleTikTokDownload(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	data, err := tiktok.FetchData(url)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	response.OK(w, data, "TikTok media fetched successfully")
}

func HandleTikTokProfile(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	data, err := tiktok.FetchUser(url)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	response.OK(w, data, "TikTok profile fetched successfully")
}
