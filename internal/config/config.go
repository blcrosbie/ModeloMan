package config

import (
	"os"
)

type Config struct {
	GRPCAddr  string
	DataFile  string
	AuthToken string
}

func Load() Config {
	return Config{
		GRPCAddr:  envOrDefault("GRPC_ADDR", ":50051"),
		DataFile:  envOrDefault("DATA_FILE", "./data/modeloman.db.json"),
		AuthToken: os.Getenv("AUTH_TOKEN"),
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
