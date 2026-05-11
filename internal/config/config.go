package config

import "os"

type Config struct {
	Port string
	Host string
}

func Load() Config {
	return Config{
		Port: getEnv("PORT", "3000"),
		Host: getEnv("HOST", "0.0.0.0"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
