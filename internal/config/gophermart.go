package config

import "fmt"

type GophermartConfig struct {
	RunAddress           string
	DB                   *DBConfig
	AccrualSystemAddress string
}

func LoadGophermartConfig(runAddrFlag, dbFlag, accrualAddrFlag string) (*GophermartConfig, error) {
	runAddr, err := GetEnvOrValue(runAddrFlag, "RUN_ADDRESS")
	if err != nil {
		return nil, fmt.Errorf("failed to load RUN_ADDRESS: %w", err)
	}

	dbCfg, err := LoadDBConfig(dbFlag)
	if err != nil {
		return nil, fmt.Errorf("failed to load DB config: %w", err)
	}

	accrualAddr, err := GetEnvOrValue(accrualAddrFlag, "ACCRUAL_SYSTEM_ADDRESS")
	if err != nil {
		return nil, fmt.Errorf("failed to load ACCRUAL_SYSTEM_ADDRESS: %w", err)
	}

	return &GophermartConfig{
		RunAddress:           runAddr,
		DB:                   dbCfg,
		AccrualSystemAddress: accrualAddr,
	}, nil
}
