package handler

import (
	"net/http"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/platform/spotify"
	"github.com/pavelc4/astra/internal/response"
)

func HandleSpotifyDownload(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	data, err := spotify.FetchData(url)
	if err != nil {
		response.HandleError(w, err)
		return
	}

	response.OK(w, data, "Spotify media fetched successfully")
}
