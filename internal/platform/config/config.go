package config

import "fmt"

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
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	Name        string `yaml:"name"`
	User        string `yaml:"user"`
	Password    string `yaml:"password"`
	AutoMigrate bool   `yaml:"auto_migrate"`
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		p.Host, p.Port, p.User, p.Password, p.Name,
	)
}

type RabbitMQConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Vhost    string `yaml:"vhost"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func (r RabbitMQConfig) URL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
		r.User, r.Password, r.Host, r.Port, r.Vhost,
	)
}
