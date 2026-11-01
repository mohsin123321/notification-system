package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Gateway  GatewayConfig  `yaml:"gateway"`
	Notifier NotifierConfig `yaml:"notifier"`
	Users    UsersConfig    `yaml:"users"`
	Postgres PostgresConfig `yaml:"postgres"`
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
}

type GatewayConfig struct {
	Addr string `yaml:"addr"`
}

type NotifierConfig struct {
	GRPCAddr string `yaml:"grpc_addr"`
}

type UsersConfig struct {
	GRPCAddr string `yaml:"grpc_addr"`
}

type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

type RabbitMQConfig struct {
	URL string `yaml:"url"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Default() *Config {
	return &Config{
		Gateway:  GatewayConfig{Addr: ":8080"},
		Notifier: NotifierConfig{GRPCAddr: ":9090"},
		Users:    UsersConfig{GRPCAddr: ":9091"},
		Postgres: PostgresConfig{DSN: "postgres://notify:notify@localhost:5433/notifications?sslmode=disable"},
		RabbitMQ: RabbitMQConfig{URL: "amqp://notify:notify@localhost:5672/"},
	}
}
