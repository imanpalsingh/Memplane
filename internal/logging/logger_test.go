package logging

import "testing"

func TestNewAcceptsSupportedEnvironments(t *testing.T) {
	environments := []string{"production", "development", "test"}
	for _, env := range environments {
		logger, err := New(env, "info")
		if err != nil {
			t.Fatalf("expected no error for environment %q, got %v", env, err)
		}
		_ = logger.Sync()
	}
}

func TestNewRejectsInvalidLogLevel(t *testing.T) {
	_, err := New("production", "invalid")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
