package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.Addr)
	}
	if cfg.MaxBodyBytes != 10*1024*1024 {
		t.Fatalf("MaxBodyBytes = %d, want 10MiB", cfg.MaxBodyBytes)
	}
	if cfg.MaxDelay != 10*time.Second {
		t.Fatalf("MaxDelay = %s, want 10s", cfg.MaxDelay)
	}
	if !cfg.EnableCompression {
		t.Fatal("EnableCompression = false, want true")
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("ADDR", ":9090")
	t.Setenv("READ_TIMEOUT", "1s")
	t.Setenv("WRITE_TIMEOUT", "2s")
	t.Setenv("IDLE_TIMEOUT", "3s")
	t.Setenv("SHUTDOWN_TIMEOUT", "4s")
	t.Setenv("HANDLER_TIMEOUT", "5s")
	t.Setenv("MAX_BODY_BYTES", "2KiB")
	t.Setenv("MAX_DELAY", "6s")
	t.Setenv("MAX_STREAM_ITEMS", "7")
	t.Setenv("MAX_RANDOM_BYTES", "8")
	t.Setenv("TRUST_PROXY_HEADERS", "true")
	t.Setenv("ENABLE_COMPRESSION", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Addr != ":9090" || cfg.ReadTimeout != time.Second || cfg.WriteTimeout != 2*time.Second {
		t.Fatalf("unexpected timeout overrides: %+v", cfg)
	}
	if cfg.MaxBodyBytes != 2048 || cfg.MaxDelay != 6*time.Second {
		t.Fatalf("unexpected limit overrides: %+v", cfg)
	}
	if cfg.MaxStreamItems != 7 || cfg.MaxRandomBytes != 8 {
		t.Fatalf("unexpected count overrides: %+v", cfg)
	}
	if !cfg.TrustProxyHeaders || cfg.EnableCompression {
		t.Fatalf("unexpected boolean overrides: %+v", cfg)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := map[string]string{
		"READ_TIMEOUT":     "soon",
		"MAX_BODY_BYTES":   "large",
		"MAX_STREAM_ITEMS": "0",
		"MAX_RANDOM_BYTES": "0",
	}

	for key, value := range tests {
		t.Run(key, func(t *testing.T) {
			t.Setenv(key, value)
			if _, err := Load(); err == nil {
				t.Fatal("Load() error = nil, want error")
			}
		})
	}
}
