package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/lmittmann/tint"
	"github.com/pavelc4/astra/internal/config"
)

func main() {
	cfg := config.Load()

	slog.SetDefault(slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		TimeFormat: time.StampMilli,
	})))

	slog.Info("cookies check", 
		"instagram_loaded", os.Getenv("INSTAGRAM_COOKIES") != "",
		"facebook_loaded", os.Getenv("FACEBOOK_COOKIES") != "",
	)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger)
	r.Use(middleware.Recoverer)

	registerRoutes(r)

	addr := cfg.Host + ":" + cfg.Port
	slog.Info("server starting", "port", cfg.Port)
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
