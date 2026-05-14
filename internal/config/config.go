package config

import (
	"bufio"
	"os"
	"strings"

	"github.com/pavelc4/astra/internal/instagram"
)

type Config struct {
	Port string
	Host string
}

func Load() Config {
	loadDotEnv(".env")

	cfg := Config{
		Port: getEnv("PORT", "3000"),
		Host: getEnv("HOST", "0.0.0.0"),
	}

	if c := os.Getenv("INSTAGRAM_COOKIES"); c != "" {
		instagram.SetCookies(c)
	}

	return cfg
}

// loadDotEnv reads key=value pairs from a .env file and sets them as
// environment variables — skips keys that are already set in the environment
// so that real env vars always take precedence.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // nggak ada .env? fine, skip aja
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines dan komentar
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}

		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
