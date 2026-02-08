package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultHTTPAddr          = ":8080"
	defaultShutdownTimeout   = 10 * time.Second
	defaultReadHeaderTimeout = 5 * time.Second
	defaultWriteTimeout      = 15 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultLogLevel          = "info"
	defaultEnvironment       = "production"
)

type Config struct {
	HTTPAddr          string
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	LogLevel          string
	Environment       string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, err
	}

	cfg := Config{
		HTTPAddr:          defaultHTTPAddr,
		ShutdownTimeout:   defaultShutdownTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		LogLevel:          defaultLogLevel,
		Environment:       defaultEnvironment,
	}

	if v := strings.TrimSpace(os.Getenv("MEMPLANE_HTTP_ADDR")); v != "" {
		cfg.HTTPAddr = v
	}

	if d, ok, err := readDurationEnv("MEMPLANE_SHUTDOWN_TIMEOUT"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.ShutdownTimeout = d
	}

	if d, ok, err := readDurationEnv("MEMPLANE_READ_HEADER_TIMEOUT"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.ReadHeaderTimeout = d
	}

	if d, ok, err := readDurationEnv("MEMPLANE_WRITE_TIMEOUT"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.WriteTimeout = d
	}

	if d, ok, err := readDurationEnv("MEMPLANE_IDLE_TIMEOUT"); err != nil {
		return Config{}, err
	} else if ok {
		cfg.IdleTimeout = d
	}

	if v := strings.TrimSpace(os.Getenv("MEMPLANE_LOG_LEVEL")); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}

	if v := strings.TrimSpace(os.Getenv("MEMPLANE_ENV")); v != "" {
		cfg.Environment = strings.ToLower(v)
	}

	switch cfg.Environment {
	case "production", "development", "test":
	default:
		return Config{}, fmt.Errorf("MEMPLANE_ENV must be one of: production, development, test")
	}

	return cfg, nil
}

func readDurationEnv(key string) (time.Duration, bool, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0, false, nil
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, false, fmt.Errorf("parse %s: %w", key, err)
	}
	if d <= 0 {
		return 0, false, fmt.Errorf("%s must be positive", key)
	}

	return d, true, nil
}
