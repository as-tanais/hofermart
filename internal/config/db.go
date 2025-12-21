package config

import (
	"fmt"
)

type DBConfig struct {
	DatabaseURI string
}

func LoadDBConfig(dbFlag string) (*DBConfig, error) {
	uri, err := GetEnvOrValue(dbFlag, "DATABASE_URI")
	if err != nil {
		return nil, fmt.Errorf("failed to load DB config: %w", err)
	}
	return &DBConfig{DatabaseURI: uri}, nil
}
