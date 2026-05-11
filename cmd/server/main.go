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

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		handler.OK(w, map[string]any{
			"endpoints": []string{
				"/api/bluesky/download",
				"/api/capcut/download",
				"/api/dailymotion/download",
				"/api/douyin/download",
				"/api/kuaishou/download",
				"/api/linkedin/download",
				"/api/meta/download",
				"/api/pinterest/download",
				"/api/reddit/download",
				"/api/snapchat/download",
				"/api/soundcloud/download",
				"/api/spotify/download",
				"/api/terabox/download",
				"/api/threads/download",
				"/api/tiktok/download",
				"/api/tumblr/download",
				"/api/twitter/download",
			},
		}, "Universal Downloader API is running")
	})

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	slog.Info("server starting", "addr", addr)
	http.ListenAndServe(addr, r)
}
