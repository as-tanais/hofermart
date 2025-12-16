package config

import (
	"flag"
	"fmt"
)

type AccrualConfig struct {
	Address string
	DBConfig
}

func NewAccrualConfig() (*AccrualConfig, error) {
	cfg := &AccrualConfig{}

	addrFlag := flag.String("a", "localhost:8081", "Accrual server address host:port")

	flag.Parse()

	cfg.Address = GetEnvOrDefault("RUN_ADDRESS", *addrFlag)

	if cfg.Address == "" {
		return nil, fmt.Errorf("server address cannot be empty")
	}

	dbURI, err := getDBURI()

	if err != nil {
		return nil, err
	}

	cfg.DATABASE_URI = dbURI

	return cfg, nil

}
