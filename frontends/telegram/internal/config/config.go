package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds configuration for the Telegram bot.
type Config struct {
	BotToken       string
	RootTelegramID int64
	CoreGRPCAddr   string
	WhisperAddr    string // optional; if empty, voice/video notes are not transcribed
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		BotToken:     getEnv("BOT_TOKEN", ""),
		CoreGRPCAddr: getEnv("CORE_GRPC_ADDR", "localhost:50051"),
		WhisperAddr:  getEnv("WHISPER_GRPC_ADDR", ""),
	}
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}

	rootIDStr := os.Getenv("ROOT_TELEGRAM_ID")
	if rootIDStr == "" {
		return nil, fmt.Errorf("ROOT_TELEGRAM_ID is required")
	}
	rootID, err := strconv.ParseInt(rootIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid ROOT_TELEGRAM_ID: %w", err)
	}
	cfg.RootTelegramID = rootID

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
