package config

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var configFlag = flag.String("c", "config.local.yaml", "path to configuration file")

// LoadConfigFromFlag reads the config path from the -c CLI flag.
func LoadConfigFromFlag() (*Config, error) {
	if !flag.Parsed() {
		flag.Parse()
	}
	return LoadConfig(*configFlag)
}

// LoadConfig reads a YAML config file, loads .env if present, and applies env var overrides.
func LoadConfig(path string) (*Config, error) {
	loadDotEnv(dotEnvPath(path))

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	applyEnvOverrides(&cfg)
	return &cfg, nil
}

// dotEnvPath derives the .env path from the config file's directory.
func dotEnvPath(configPath string) string {
	dir := configPath
	if idx := strings.LastIndexAny(dir, "/\\"); idx >= 0 {
		dir = dir[:idx]
	} else {
		dir = "."
	}
	return dir + "/.env"
}

// loadDotEnv reads a .env file and sets env vars not already set by the environment.
// Silently does nothing if the file does not exist.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		// Real environment variables take precedence over .env values.
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value) //nolint:errcheck
		}
	}
}

// applyEnvOverrides applies NOTIFY_* environment variables on top of YAML values.
// Non-sensitive settings live in YAML; secrets (passwords, credentials) go in .env.
func applyEnvOverrides(cfg *Config) {
	// Gateway
	if v := os.Getenv("NOTIFY_GATEWAY_ADDR"); v != "" {
		cfg.Gateway.Addr = v
	}

	// Notifier
	if v := os.Getenv("NOTIFY_NOTIFIER_GRPC_ADDR"); v != "" {
		cfg.Notifier.GRPCAddr = v
	}

	// Users
	if v := os.Getenv("NOTIFY_USERS_GRPC_ADDR"); v != "" {
		cfg.Users.GRPCAddr = v
	}

	// Postgres — host/port/name in YAML, user/password in .env
	if v := os.Getenv("NOTIFY_POSTGRES_HOST"); v != "" {
		cfg.Postgres.Host = v
	}
	if v := os.Getenv("NOTIFY_POSTGRES_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Postgres.Port = port
		}
	}
	if v := os.Getenv("NOTIFY_POSTGRES_NAME"); v != "" {
		cfg.Postgres.Name = v
	}
	if v := os.Getenv("NOTIFY_POSTGRES_USER"); v != "" {
		cfg.Postgres.User = v
	}
	if v := os.Getenv("NOTIFY_POSTGRES_PASSWORD"); v != "" {
		cfg.Postgres.Password = v
	}

	// RabbitMQ — host/port/vhost in YAML, user/password in .env
	if v := os.Getenv("NOTIFY_RABBITMQ_HOST"); v != "" {
		cfg.RabbitMQ.Host = v
	}
	if v := os.Getenv("NOTIFY_RABBITMQ_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.RabbitMQ.Port = port
		}
	}
	if v := os.Getenv("NOTIFY_RABBITMQ_VHOST"); v != "" {
		cfg.RabbitMQ.Vhost = v
	}
	if v := os.Getenv("NOTIFY_RABBITMQ_USER"); v != "" {
		cfg.RabbitMQ.User = v
	}
	if v := os.Getenv("NOTIFY_RABBITMQ_PASSWORD"); v != "" {
		cfg.RabbitMQ.Password = v
	}
}
