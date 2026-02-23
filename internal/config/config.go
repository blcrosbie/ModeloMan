package config

import (
	"os"
	"strconv"
)

type Config struct {
	GRPCAddr          string
	HTTPAddr          string
	StoreDriver       string
	DataFile          string
	DatabaseURL       string
	AuthToken         string
	AllowLegacyAuth   bool
	EnableReflection  bool
	BootstrapAgentID  string
	BootstrapAgentKey string
}

func Load() Config {
	return Config{
		GRPCAddr:          envOrDefault("GRPC_ADDR", "127.0.0.1:50051"),
		HTTPAddr:          envOrDefault("HTTP_ADDR", "127.0.0.1:8080"),
		StoreDriver:       envOrDefault("STORE_DRIVER", "file"),
		DataFile:          envOrDefault("DATA_FILE", "./data/modeloman.db.json"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		AuthToken:         os.Getenv("AUTH_TOKEN"),
		AllowLegacyAuth:   envBoolOrDefault("ALLOW_LEGACY_AUTH_TOKEN", false),
		EnableReflection:  envBoolOrDefault("ENABLE_REFLECTION", false),
		BootstrapAgentID:  envOrDefault("BOOTSTRAP_AGENT_ID", "orchestrator"),
		BootstrapAgentKey: os.Getenv("BOOTSTRAP_AGENT_KEY"),
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envBoolOrDefault(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
