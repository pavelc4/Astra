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

	data, err := instagram.FetchStories(username)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	response.OK(w, data, "Instagram stories fetched successfully")
}

func HandleInstagramProfile(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	data, err := meta.FetchInstagramProfile(url)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	response.OK(w, data, "Instagram profile fetched successfully")
}

func HandleInstagramDownload(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	data, err := meta.FetchInstagramData(url)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	response.OK(w, data, "Instagram media fetched successfully")
}
