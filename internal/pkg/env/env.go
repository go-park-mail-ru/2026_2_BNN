package env

import (
	"log/slog"
	"os"
)

func Must(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("Environment variable is not set", "key", key)
		os.Exit(1)
	}

	return value
}
