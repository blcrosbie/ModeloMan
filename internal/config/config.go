package config

import (
	"os"
)

type Config struct {
	GRPCAddr          string
	HTTPAddr          string
	StoreDriver       string
	DataFile          string
	DatabaseURL       string
	AuthToken         string
	BootstrapAgentID  string
	BootstrapAgentKey string
}

func Load() Config {
	return Config{
		GRPCAddr:          envOrDefault("GRPC_ADDR", ":50051"),
		HTTPAddr:          envOrDefault("HTTP_ADDR", ":8080"),
		StoreDriver:       envOrDefault("STORE_DRIVER", "file"),
		DataFile:          envOrDefault("DATA_FILE", "./data/modeloman.db.json"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		AuthToken:         os.Getenv("AUTH_TOKEN"),
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
