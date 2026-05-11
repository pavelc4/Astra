package handler

import (
	"net/http"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/platform/meta"
	"github.com/pavelc4/astra/internal/response"
)

func HandleFacebookDownload(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	data, err := meta.FetchFacebookData(url)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	response.OK(w, data, "Facebook media fetched successfully")
}
