package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTP            HTTPConfig
	Database        DatabaseConfig
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
}

type HTTPConfig struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
}

func Load() (Config, error) {
	httpAddr, err := requiredString("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}

	logLevel, err := parseLogLevel("LOG_LEVEL")
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := positiveDuration("SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	httpReadTimeout, err := positiveDuration("HTTP_READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	httpReadHeaderTimeout, err := positiveDuration("HTTP_READ_HEADER_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	httpWriteTimeout, err := positiveDuration("HTTP_WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	httpIdleTimeout, err := positiveDuration("HTTP_IDLE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	databaseURL, err := requiredString("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	databaseMaxConns, err := positiveInt32("DATABASE_MAX_CONNS")
	if err != nil {
		return Config{}, err
	}

	databaseMinConns, err := nonNegativeInt32("DATABASE_MIN_CONNS")
	if err != nil {
		return Config{}, err
	}

	if databaseMinConns > databaseMaxConns {
		return Config{}, fmt.Errorf(
			"DATABASE_MIN_CONNS must not exceed DATABASE_MAX_CONNS",
		)
	}

	databaseMaxConnLifetime, err := positiveDuration(
		"DATABASE_MAX_CONN_LIFETIME",
	)
	if err != nil {
		return Config{}, err
	}

	databaseConnectTimeout, err := positiveDuration(
		"DATABASE_CONNECT_TIMEOUT",
	)
	if err != nil {
		return Config{}, err
	}

	databaseQueryTimeout, err := positiveDuration(
		"DATABASE_QUERY_TIMEOUT",
	)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTP: HTTPConfig{
			Addr:              httpAddr,
			ReadTimeout:       httpReadTimeout,
			ReadHeaderTimeout: httpReadHeaderTimeout,
			WriteTimeout:      httpWriteTimeout,
			IdleTimeout:       httpIdleTimeout,
		},
		Database: DatabaseConfig{
			URL:             databaseURL,
			MaxConns:        databaseMaxConns,
			MinConns:        databaseMinConns,
			MaxConnLifetime: databaseMaxConnLifetime,
			ConnectTimeout:  databaseConnectTimeout,
			QueryTimeout:    databaseQueryTimeout,
		},
		LogLevel:        logLevel,
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

func requiredString(name string) (string, error) {
	value, ok := os.LookupEnv(name)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	return value, nil
}

func positiveDuration(name string) (time.Duration, error) {
	value, err := requiredString(name)
	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", name, err)
	}

	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return duration, nil
}

func positiveInt32(name string) (int32, error) {
	value, err := parseInt32(name)
	if err != nil {
		return 0, err
	}

	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return value, nil
}

func nonNegativeInt32(name string) (int32, error) {
	value, err := parseInt32(name)
	if err != nil {
		return 0, err
	}

	if value < 0 {
		return 0, fmt.Errorf("%s must not be negative", name)
	}

	return value, nil
}

func parseInt32(name string) (int32, error) {
	rawValue, err := requiredString(name)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseInt(rawValue, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer: %w", name, err)
	}

	return int32(value), nil
}

func parseLogLevel(name string) (slog.Level, error) {
	value, err := requiredString(name)
	if err != nil {
		return 0, err
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(value)); err != nil {
		return 0, fmt.Errorf("%s contains an invalid log level: %w", name, err)
	}

	return level, nil
}
