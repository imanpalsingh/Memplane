package logging

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(environment, level string) (*zap.Logger, error) {
	cfg, err := zapConfig(environment)
	if err != nil {
		return nil, err
	}

	lvl := zap.NewAtomicLevel()
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return nil, err
	}
	cfg.Level = lvl
	cfg.EncoderConfig.TimeKey = "time"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return cfg.Build()
}

func zapConfig(environment string) (zap.Config, error) {
	switch environment {
	case "production", "test":
		return zap.NewProductionConfig(), nil
	case "development":
		return zap.NewDevelopmentConfig(), nil
	default:
		return zap.Config{}, fmt.Errorf("unsupported environment: %s", environment)
	}
}
