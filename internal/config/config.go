package config

import (
	"os"

	"github.com/pavelc4/astra/internal/instagram"
)

type Config struct {
	Port string
	Host string
}

func Load() Config {
	cfg := Config{
		Port: getEnv("PORT", "3000"),
		Host: getEnv("HOST", "0.0.0.0"),
	}

	if c := os.Getenv("INSTAGRAM_COOKIES"); c != "" {
		instagram.SetCookies(c)
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
