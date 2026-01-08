package config

import (
	"time"
)

type DBConfig struct {
	DatabaseURI string
}

type ServerConfig struct {
	RunAddress string
	DB         DBConfig
}

type AccrualConfig struct {
	ServerConfig
}

type GophermartConfig struct {
	ServerConfig
	AccrualSystemAddress string
}

type ServerConfigOptions struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func DefaultServerConfig() ServerConfigOptions {
	return ServerConfigOptions{
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}
}
