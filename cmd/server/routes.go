package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pavelc4/astra/internal/handler"
	"github.com/pavelc4/astra/internal/response"
)

func registerRoutes(r chi.Router) {
	r.Get("/health", handler.HandleHealth)

	r.Route("/api/tiktok", func(r chi.Router) {
		r.Get("/download", handler.HandleTikTokDownload)
		r.Get("/profile", handler.HandleTikTokProfile)
		r.Get("/music", handler.HandleTikTokMusic)
	})
	r.Route("/api/twitter", func(r chi.Router) {
		r.Get("/download", handler.HandleTwitterDownload)
	})
	r.Route("/api/meta", func(r chi.Router) {
		r.Get("/instagram/download", handler.HandleInstagramDownload)
		r.Get("/facebook/download", handler.HandleFacebookDownload)
		r.Get("/threads/download", handler.HandleThreadsDownload)
	})
	r.Route("/api/reddit", func(r chi.Router) {
		r.Get("/download", handler.HandleRedditDownload)
	})
	r.Route("/api/pinterest", func(r chi.Router) {
		r.Get("/download", handler.HandlePinterestDownload)
	})
	r.Route("/api/terabox", func(r chi.Router) {
		r.Get("/download", handler.HandleTeraboxDownload)
	})
	r.Route("/api/spotify", func(r chi.Router) {
		r.Get("/download", handler.HandleSpotifyDownload)
	})
	r.Route("/api/soundcloud", func(r chi.Router) {
		r.Get("/download", handler.HandleSoundcloudDownload)
	})
	r.Route("/api/capcut", func(r chi.Router) {
		r.Get("/download", handler.HandleCapcutDownload)
	})
	r.Route("/api/linkedin", func(r chi.Router) {
		r.Get("/download", handler.HandleLinkedinDownload)
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, map[string]any{
			"endpoints": []string{
				"/health",
				"/api/tiktok/download",
				"/api/tiktok/profile",
				"/api/tiktok/music",
				"/api/twitter/download",
				"/api/meta/instagram/download",
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
	})
}
