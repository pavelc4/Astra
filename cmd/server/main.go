package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lmittmann/tint"
	"github.com/pavelc4/astra/internal/config"
	"github.com/pavelc4/astra/internal/handler"
)

func main() {
	cfg := config.Load()

	slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.StampMilli,
	})))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)

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
				"/api/soundcloud/download",
				"/api/capcut/download",
				"/api/linkedin/download",
			},
		}, "Universal Downloader API is running")
	})

	endpoints := []string{
		"GET  /",
		"GET  /api/tiktok/download",
		"GET  /api/twitter/download",
		"GET  /api/meta/instagram/download",
		"GET  /api/meta/facebook/download",
		"GET  /api/meta/threads/download",
		"GET  /api/reddit/download",
		"GET  /api/pinterest/download",
		"GET  /api/terabox/download",
		"GET  /api/spotify/download",
		"GET  /api/soundcloud/download",
		"GET  /api/capcut/download",
		"GET  /api/linkedin/download",
	}

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	slog.Info("server starting", "addr", addr, "routes", len(endpoints))
	for _, ep := range endpoints {
		slog.Info("  " + ep)
	}
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server stopped", "error", err)
	}
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"duration", time.Since(start).String(),
		)
	})
}
