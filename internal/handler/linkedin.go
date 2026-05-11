package handler

import (
	"net/http"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/platform/linkedin"
	"github.com/pavelc4/astra/internal/response"
)

func HandleLinkedinDownload(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	data, err := linkedin.FetchData(url)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	response.OK(w, data, "LinkedIn media fetched successfully")
}
