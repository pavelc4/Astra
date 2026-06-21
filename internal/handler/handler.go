package handler

import (
	"context"
	"net/http"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/response"
)

// makeDownloadHandler creates a standard download handler for a given fetch function.
func makeDownloadHandler[T any](fetchFunc func(ctx context.Context, url string) (T, error), successMsg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			response.HandleError(w, errors.NewValidation("url parameter is required"))
			return
		}

		data, err := fetchFunc(r.Context(), url)
		if err != nil {
			response.HandleError(w, err)
			return
		}

		response.OK(w, data, successMsg)
	}
}
