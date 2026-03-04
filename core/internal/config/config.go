package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the core service.
type Config struct {
	DatabaseURL    string
	GRPCPort       int
	MediaDir       string
	MaxGRPCMsgSize int
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		GRPCPort:       getEnvInt("GRPC_PORT", 50051),
		MediaDir:       getEnv("MEDIA_DIR", "/data/media"),
		MaxGRPCMsgSize: getEnvInt("MAX_GRPC_MSG_SIZE", 52428800),
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultVal
}
