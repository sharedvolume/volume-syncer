package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server ServerConfig
	Sync   SyncConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type SyncConfig struct {
	DefaultTimeout time.Duration
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getDurationEnv("READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDurationEnv("WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDurationEnv("IDLE_TIMEOUT", 120*time.Second),
		},
		Sync: SyncConfig{
			DefaultTimeout: getDurationEnv("SYNC_TIMEOUT", 5*time.Minute),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	return defaultValue
}
