package config

import "fmt"

func LoadAccrualConfig(runAddrFlag, dbFlag string) (*AccrualConfig, error) {
	runAddr, err := GetEnvOrValue(runAddrFlag, "RUN_ADDRESS")
	if err != nil {
		return nil, fmt.Errorf("failed to load RUN_ADDRESS: %w", err)
	}

	dbCfg, err := LoadDBConfig(dbFlag)
	if err != nil {
		return nil, fmt.Errorf("failed to load DB config: %w", err)
	}

	return &AccrualConfig{
		ServerConfig: ServerConfig{
			RunAddress: runAddr,
			DB:         *dbCfg,
		},
	}, nil
}
