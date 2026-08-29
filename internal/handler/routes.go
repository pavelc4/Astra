package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/pavelc4/astra/internal/errors"
	"github.com/pavelc4/astra/internal/platform/capcut"
	"github.com/pavelc4/astra/internal/platform/linkedin"
	"github.com/pavelc4/astra/internal/platform/meta/facebook"
	"github.com/pavelc4/astra/internal/platform/meta/instagram"
	"github.com/pavelc4/astra/internal/platform/meta/threads"
	"github.com/pavelc4/astra/internal/platform/pinterest"
	"github.com/pavelc4/astra/internal/platform/reddit"
	"github.com/pavelc4/astra/internal/platform/soundcloud"
	"github.com/pavelc4/astra/internal/platform/spotify"
	"github.com/pavelc4/astra/internal/platform/terabox"
	"github.com/pavelc4/astra/internal/platform/tiktok"
	"github.com/pavelc4/astra/internal/platform/twitter"
	"github.com/pavelc4/astra/internal/response"
)

// Register wires every route onto r. All download endpoints share the injected
// request cache via download(h, ...); this is the single place platform routes
// live (replacing the former one-file-per-platform handlers).
func (h *Handlers) Register(r chi.Router) {
	r.Get("/health", h.Health)

	r.Route("/api/tiktok", func(r chi.Router) {
		r.Get("/download", download(h, tiktok.FetchData, "TikTok media fetched successfully"))
		r.Get("/profile", download(h, tiktok.FetchUser, "TikTok profile fetched successfully"))
		r.Get("/music", download(h, tiktok.FetchMusic, "TikTok music fetched successfully"))
	})
	r.Route("/api/twitter", func(r chi.Router) {
		r.Get("/download", download(h, twitter.FetchData, "Twitter media fetched successfully"))
	})
	r.Route("/api/meta", func(r chi.Router) {
		r.Get("/instagram/download", download(h, instagram.FetchData, "Instagram media fetched successfully"))
		r.Get("/instagram/profile", download(h, instagram.FetchProfileByURL, "Instagram profile fetched successfully"))
		r.Get("/instagram/stories", h.InstagramStories)
		r.Get("/facebook/download", download(h, facebook.FetchData, "Facebook media fetched successfully"))
		r.Get("/threads/download", download(h, threads.FetchData, "Threads media fetched successfully"))
	})
	r.Route("/api/reddit", func(r chi.Router) {
		r.Get("/download", download(h, reddit.FetchData, "Reddit media fetched successfully"))
	})
	r.Route("/api/pinterest", func(r chi.Router) {
		r.Get("/download", download(h, pinterest.FetchData, "Pinterest media fetched successfully"))
	})
	r.Route("/api/terabox", func(r chi.Router) {
		r.Get("/download", download(h, terabox.FetchData, "TeraBox media fetched successfully"))
	})
	r.Route("/api/spotify", func(r chi.Router) {
		r.Get("/download", download(h, spotify.FetchData, "Spotify media fetched successfully"))
	})
	r.Route("/api/soundcloud", func(r chi.Router) {
		r.Get("/download", download(h, soundcloud.FetchData, "SoundCloud media fetched successfully"))
	})
	r.Route("/api/capcut", func(r chi.Router) {
		r.Get("/download", download(h, capcut.FetchData, "CapCut media fetched successfully"))
	})
	r.Route("/api/linkedin", func(r chi.Router) {
		r.Get("/download", download(h, linkedin.FetchData, "LinkedIn media fetched successfully"))
	})

	r.Get("/", handleRoot)
}

// InstagramStories is a bespoke handler: stories are fetched by username, not by
// the shared url→media flow, so it doesn't go through download().
func (h *Handlers) InstagramStories(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		response.HandleError(w, errors.NewValidation("url parameter is required"))
		return
	}

	username := instagram.ExtractUsername(rawURL)
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

func handleRoot(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]any{
		"endpoints": []string{
			"/health",
			"/api/tiktok/download",
			"/api/tiktok/profile",
			"/api/tiktok/music",
			"/api/twitter/download",
			"/api/meta/instagram/download",
			"/api/meta/instagram/profile",
			"/api/meta/instagram/stories",
			"/api/meta/facebook/download",
			"/api/meta/threads/download",
			"/api/reddit/download",
			"/api/pinterest/download",
			"/api/terabox/download",
			"/api/spotify/download",
			"/api/soundcloud/download",
			"/api/capcut/download",
			"/api/linkedin/download",
		},
	}, "Universal Downloader API is running")
}
