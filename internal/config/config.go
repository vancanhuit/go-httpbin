package config

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Addr              string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	HandlerTimeout    time.Duration
	MaxBodyBytes      int64
	MaxDelay          time.Duration
	MaxStreamItems    int
	MaxRandomBytes    int
	TrustProxyHeaders bool
	EnableCompression bool
	LogLevel          slog.Level
	LogAddSource      bool
}

type rawConfig struct {
	Addr              string `env:"ADDR" envDefault:":8080"`
	ReadTimeout       string `env:"READ_TIMEOUT" envDefault:"5s"`
	WriteTimeout      string `env:"WRITE_TIMEOUT" envDefault:"30s"`
	IdleTimeout       string `env:"IDLE_TIMEOUT" envDefault:"120s"`
	ShutdownTimeout   string `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	HandlerTimeout    string `env:"HANDLER_TIMEOUT" envDefault:"30s"`
	MaxBodyBytes      string `env:"MAX_BODY_BYTES" envDefault:"10MiB"`
	MaxDelay          string `env:"MAX_DELAY" envDefault:"10s"`
	MaxStreamItems    int    `env:"MAX_STREAM_ITEMS" envDefault:"100"`
	MaxRandomBytes    int    `env:"MAX_RANDOM_BYTES" envDefault:"1048576"`
	TrustProxyHeaders bool   `env:"TRUST_PROXY_HEADERS" envDefault:"false"`
	EnableCompression bool   `env:"ENABLE_COMPRESSION" envDefault:"true"`
	LogLevel          string `env:"LOG_LEVEL" envDefault:"info"`
	LogAddSource      bool   `env:"LOG_ADD_SOURCE" envDefault:"false"`
}

func Load() (Config, error) {
	var raw rawConfig
	if err := env.Parse(&raw); err != nil {
		return Config{}, err
	}

	readTimeout, err := parseDuration("READ_TIMEOUT", raw.ReadTimeout)
	if err != nil {
		return Config{}, err
	}
	writeTimeout, err := parseDuration("WRITE_TIMEOUT", raw.WriteTimeout)
	if err != nil {
		return Config{}, err
	}
	idleTimeout, err := parseDuration("IDLE_TIMEOUT", raw.IdleTimeout)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := parseDuration("SHUTDOWN_TIMEOUT", raw.ShutdownTimeout)
	if err != nil {
		return Config{}, err
	}
	handlerTimeout, err := parseDuration("HANDLER_TIMEOUT", raw.HandlerTimeout)
	if err != nil {
		return Config{}, err
	}
	maxDelay, err := parseDuration("MAX_DELAY", raw.MaxDelay)
	if err != nil {
		return Config{}, err
	}
	maxBodyBytes, err := parseBytes("MAX_BODY_BYTES", raw.MaxBodyBytes)
	if err != nil {
		return Config{}, err
	}
	if raw.MaxStreamItems < 1 {
		return Config{}, fmt.Errorf("MAX_STREAM_ITEMS must be greater than 0")
	}
	if raw.MaxRandomBytes < 1 {
		return Config{}, fmt.Errorf("MAX_RANDOM_BYTES must be greater than 0")
	}
	logLevel, err := parseLogLevel("LOG_LEVEL", raw.LogLevel)
	if err != nil {
		return Config{}, err
	}

	return Config{
		Addr:              raw.Addr,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		ShutdownTimeout:   shutdownTimeout,
		HandlerTimeout:    handlerTimeout,
		MaxBodyBytes:      maxBodyBytes,
		MaxDelay:          maxDelay,
		MaxStreamItems:    raw.MaxStreamItems,
		MaxRandomBytes:    raw.MaxRandomBytes,
		TrustProxyHeaders: raw.TrustProxyHeaders,
		EnableCompression: raw.EnableCompression,
		LogLevel:          logLevel,
		LogAddSource:      raw.LogAddSource,
	}, nil
}

func parseDuration(name, value string) (time.Duration, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a Go duration: %w", name, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("%s must not be negative", name)
	}
	return d, nil
}

func parseBytes(name, value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("%s must not be empty", name)
	}

	units := []struct {
		suffix string
		size   int64
	}{
		{"tib", 1024 * 1024 * 1024 * 1024},
		{"gib", 1024 * 1024 * 1024},
		{"mib", 1024 * 1024},
		{"kib", 1024},
		{"tb", 1000 * 1000 * 1000 * 1000},
		{"gb", 1000 * 1000 * 1000},
		{"mb", 1000 * 1000},
		{"kb", 1000},
		{"b", 1},
	}

	lower := strings.ToLower(trimmed)
	for _, unit := range units {
		if strings.HasSuffix(lower, unit.suffix) {
			number := strings.TrimSpace(trimmed[:len(trimmed)-len(unit.suffix)])
			return parseByteNumber(name, number, unit.size)
		}
	}

	return parseByteNumber(name, trimmed, 1)
}

func parseByteNumber(name, value string, multiplier int64) (int64, error) {
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a byte size: %w", name, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("%s must not be negative", name)
	}
	return int64(n * float64(multiplier)), nil
}

func parseLogLevel(name, value string) (slog.Level, error) {
	var level slog.Level
	if err := level.UnmarshalText([]byte(strings.ToLower(strings.TrimSpace(value)))); err != nil {
		return 0, fmt.Errorf("%s must be one of debug, info, warn, or error: %w", name, err)
	}
	return level, nil
}
