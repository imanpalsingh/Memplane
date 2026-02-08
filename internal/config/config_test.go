package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	setEnv(t, "MEMPLANE_HTTP_ADDR", "")
	setEnv(t, "MEMPLANE_SHUTDOWN_TIMEOUT", "")
	setEnv(t, "MEMPLANE_READ_HEADER_TIMEOUT", "")
	setEnv(t, "MEMPLANE_WRITE_TIMEOUT", "")
	setEnv(t, "MEMPLANE_IDLE_TIMEOUT", "")
	setEnv(t, "MEMPLANE_LOG_LEVEL", "")
	setEnv(t, "MEMPLANE_ENV", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("expected default addr %q, got %q", defaultHTTPAddr, cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != defaultShutdownTimeout {
		t.Fatalf("expected default timeout %v, got %v", defaultShutdownTimeout, cfg.ShutdownTimeout)
	}
	if cfg.ReadHeaderTimeout != defaultReadHeaderTimeout {
		t.Fatalf("expected default read header timeout %v, got %v", defaultReadHeaderTimeout, cfg.ReadHeaderTimeout)
	}
	if cfg.WriteTimeout != defaultWriteTimeout {
		t.Fatalf("expected default write timeout %v, got %v", defaultWriteTimeout, cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != defaultIdleTimeout {
		t.Fatalf("expected default idle timeout %v, got %v", defaultIdleTimeout, cfg.IdleTimeout)
	}
	if cfg.LogLevel != defaultLogLevel {
		t.Fatalf("expected default log level %q, got %q", defaultLogLevel, cfg.LogLevel)
	}
	if cfg.Environment != defaultEnvironment {
		t.Fatalf("expected default environment %q, got %q", defaultEnvironment, cfg.Environment)
	}
}

func TestLoadFromEnv(t *testing.T) {
	setEnv(t, "MEMPLANE_HTTP_ADDR", ":9090")
	setEnv(t, "MEMPLANE_SHUTDOWN_TIMEOUT", "5s")
	setEnv(t, "MEMPLANE_READ_HEADER_TIMEOUT", "2s")
	setEnv(t, "MEMPLANE_WRITE_TIMEOUT", "20s")
	setEnv(t, "MEMPLANE_IDLE_TIMEOUT", "30s")
	setEnv(t, "MEMPLANE_LOG_LEVEL", "debug")
	setEnv(t, "MEMPLANE_ENV", "development")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.HTTPAddr != ":9090" {
		t.Fatalf("expected addr %q, got %q", ":9090", cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != 5*time.Second {
		t.Fatalf("expected timeout %v, got %v", 5*time.Second, cfg.ShutdownTimeout)
	}
	if cfg.ReadHeaderTimeout != 2*time.Second {
		t.Fatalf("expected read header timeout %v, got %v", 2*time.Second, cfg.ReadHeaderTimeout)
	}
	if cfg.WriteTimeout != 20*time.Second {
		t.Fatalf("expected write timeout %v, got %v", 20*time.Second, cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 30*time.Second {
		t.Fatalf("expected idle timeout %v, got %v", 30*time.Second, cfg.IdleTimeout)
	}
	if cfg.LogLevel != "debug" {
		t.Fatalf("expected log level %q, got %q", "debug", cfg.LogLevel)
	}
	if cfg.Environment != "development" {
		t.Fatalf("expected environment %q, got %q", "development", cfg.Environment)
	}
}

func TestLoadRejectsInvalidTimeout(t *testing.T) {
	setEnv(t, "MEMPLANE_SHUTDOWN_TIMEOUT", "bad")
	_, err := Load()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadRejectsNonPositiveTimeout(t *testing.T) {
	setEnv(t, "MEMPLANE_SHUTDOWN_TIMEOUT", "0s")
	_, err := Load()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadRejectsInvalidReadHeaderTimeout(t *testing.T) {
	setEnv(t, "MEMPLANE_READ_HEADER_TIMEOUT", "invalid")
	_, err := Load()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadRejectsInvalidWriteTimeout(t *testing.T) {
	setEnv(t, "MEMPLANE_WRITE_TIMEOUT", "invalid")
	_, err := Load()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadRejectsInvalidIdleTimeout(t *testing.T) {
	setEnv(t, "MEMPLANE_IDLE_TIMEOUT", "invalid")
	_, err := Load()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestLoadRejectsInvalidEnvironment(t *testing.T) {
	setEnv(t, "MEMPLANE_ENV", "staging")
	_, err := Load()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func setEnv(t *testing.T, key, value string) {
	t.Helper()

	original, ok := os.LookupEnv(key)
	if value == "" {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	} else if err := os.Setenv(key, value); err != nil {
		t.Fatalf("set %s: %v", key, err)
	}

	t.Cleanup(func() {
		if ok {
			_ = os.Setenv(key, original)
			return
		}
		_ = os.Unsetenv(key)
	})
}
