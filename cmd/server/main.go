package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pavelc4/astra/internal/config"
	"github.com/pavelc4/astra/internal/handler"
)

func main() {
	cfg := config.Load()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	r.Route("/api/tiktok", func(r chi.Router) {
		r.Get("/download", handler.HandleTikTokDownload)
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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		handler.OK(w, map[string]any{
			"endpoints": []string{
				"/api/tiktok/download",
				"/api/twitter/download",
				"/api/meta/instagram/download",
				"/api/meta/facebook/download",
				"/api/meta/threads/download",
				"/api/reddit/download",
				"/api/pinterest/download",
				"/api/terabox/download",
				"/api/spotify/download",
			},
		}, "Universal Downloader API is running")
	})

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	slog.Info("server starting", "addr", addr)
	http.ListenAndServe(addr, r)
}
