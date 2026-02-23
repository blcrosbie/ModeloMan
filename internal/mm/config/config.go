package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultConfigRelPath = ".config/modeloman/mm.yaml"
)

type Config struct {
	GRPCAddr            string        `yaml:"grpc_addr"`
	GRPCInsecure        bool          `yaml:"grpc_insecure"`
	TokenEnvVar         string        `yaml:"token_env_var"`
	DefaultBackend      string        `yaml:"default_backend"`
	RedactionEnabled    bool          `yaml:"redaction"`
	MaxContextBytes     int           `yaml:"max_context_bytes"`
	MaxTranscriptBytes  int           `yaml:"max_transcript_bytes"`
	AllowRawTranscript  bool          `yaml:"allow_raw_transcript"`
	CustomRedactRegexes []string      `yaml:"custom_redaction_regex"`
	ConnectTimeout      time.Duration `yaml:"-"`
	RequestTimeout      time.Duration `yaml:"-"`
	RetryAttempts       int           `yaml:"-"`
}

func Default() Config {
	return Config{
		GRPCAddr:           "grpc.modeloman.com:443",
		GRPCInsecure:       false,
		TokenEnvVar:        "MODEL0MAN_TOKEN",
		DefaultBackend:     "codex",
		RedactionEnabled:   true,
		MaxContextBytes:    350000,
		MaxTranscriptBytes: 200000,
		AllowRawTranscript: false,
		ConnectTimeout:     8 * time.Second,
		RequestTimeout:     10 * time.Second,
		RetryAttempts:      3,
	}
}

func Load() (Config, string, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, "", err
	}

	if raw, readErr := os.ReadFile(path); readErr == nil {
		if parseErr := parseConfig(string(raw), &cfg); parseErr != nil {
			return cfg, path, fmt.Errorf("parse mm config %s: %w", path, parseErr)
		}
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return cfg, path, fmt.Errorf("read mm config %s: %w", path, readErr)
	}

	if strings.TrimSpace(cfg.GRPCAddr) == "" {
		cfg.GRPCAddr = Default().GRPCAddr
	}
	if strings.TrimSpace(cfg.TokenEnvVar) == "" {
		cfg.TokenEnvVar = Default().TokenEnvVar
	}
	if strings.TrimSpace(cfg.DefaultBackend) == "" {
		cfg.DefaultBackend = Default().DefaultBackend
	}
	if cfg.MaxContextBytes <= 0 {
		cfg.MaxContextBytes = Default().MaxContextBytes
	}
	if cfg.MaxTranscriptBytes <= 0 {
		cfg.MaxTranscriptBytes = Default().MaxTranscriptBytes
	}
	if cfg.RetryAttempts <= 0 {
		cfg.RetryAttempts = Default().RetryAttempts
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = Default().ConnectTimeout
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = Default().RequestTimeout
	}

	return cfg, path, nil
}

func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, defaultConfigRelPath), nil
}

func ResolveToken(cfg Config) string {
	name := strings.TrimSpace(cfg.TokenEnvVar)
	if name != "" {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	// Accept MODELOMAN_TOKEN for convenience if config keeps MODEL0MAN_TOKEN default.
	if value := strings.TrimSpace(os.Getenv("MODELOMAN_TOKEN")); value != "" {
		return value
	}
	return ""
}

func EnsureConfigDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func parseConfig(raw string, cfg *Config) error {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	currentListKey := ""
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "- ") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			value = trimQuotes(value)
			if currentListKey == "custom_redaction_regex" && value != "" {
				cfg.CustomRedactRegexes = append(cfg.CustomRedactRegexes, value)
			}
			continue
		}

		currentListKey = ""
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if value == "" {
			currentListKey = key
			continue
		}
		value = trimQuotes(value)
		switch key {
		case "grpc_addr":
			cfg.GRPCAddr = value
		case "grpc_insecure":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("grpc_insecure: %w", err)
			}
			cfg.GRPCInsecure = parsed
		case "token_env_var":
			cfg.TokenEnvVar = value
		case "default_backend":
			cfg.DefaultBackend = value
		case "redaction":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("redaction: %w", err)
			}
			cfg.RedactionEnabled = parsed
		case "max_context_bytes":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("max_context_bytes: %w", err)
			}
			cfg.MaxContextBytes = parsed
		case "allow_raw_transcript":
			parsed, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("allow_raw_transcript: %w", err)
			}
			cfg.AllowRawTranscript = parsed
		case "max_transcript_bytes":
			parsed, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("max_transcript_bytes: %w", err)
			}
			cfg.MaxTranscriptBytes = parsed
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func trimQuotes(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
