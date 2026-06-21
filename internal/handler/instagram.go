package handler

import (
	"net/http"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/instagram"
	"github.com/pavelc4/astra/internal/platform/meta"
	"github.com/pavelc4/astra/internal/response"
)

func HandleInstagramStories(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	username := instagram.ExtractUsername(url)
	if username == "" {
		response.HandleError(w, errors.NewValidation("could not extract username from URL"))
		return
	}

	data, err := instagram.FetchStories(r.Context(), username)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	response.OK(w, data, "Instagram stories fetched successfully")
}

var HandleInstagramProfile = makeDownloadHandler(meta.FetchInstagramProfile, "Instagram profile fetched successfully")
var HandleInstagramDownload = makeDownloadHandler(meta.FetchInstagramData, "Instagram media fetched successfully")
