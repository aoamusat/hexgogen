package configs

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Env   string
	DbURL string
	Port  string
}

// LoadConfig loads optional local configuration before reading environment values.
// Existing environment variables take precedence over .env values. Production
// deployments should set ENV=production in their environment to skip .env loading.
func LoadConfig() (Config, error) {
	if os.Getenv("ENV") != PRODUCTION {
		if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("load .env: %w", err)
		}
	}
	cfg := Config{Env: os.Getenv("ENV"), DbURL: os.Getenv("DATABASE_URL"), Port: os.Getenv("PORT")}
	if cfg.Env == "" {
		cfg.Env = DEVELOPMENT
	}
	if cfg.Port == "" {
		cfg.Port = "8000"
	}
	if cfg.Env != DEVELOPMENT && cfg.Env != PRODUCTION {
		return Config{}, fmt.Errorf("ENV must be %q or %q", DEVELOPMENT, PRODUCTION)
	}
	if cfg.DbURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, errors.New("PORT must be a number between 1 and 65535")
	}
	return cfg, nil
}
