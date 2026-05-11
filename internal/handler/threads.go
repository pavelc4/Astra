package handler

import (
	"net/http"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/platform/meta"
)

func HandleThreadsDownload(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	data, err := meta.FetchThreadsData(url)
	if err != nil {
		HandleError(w, err)
		return
	}

	OK(w, data, "Threads media fetched successfully")
}
