package config

import (
	"fmt"
	"time"
)

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

	jwtSecret, err := GetEnvOrValue("My-strong-secret-for-JWT-bla-blab-123", "JWT_SECRET")
	if err != nil {
		return nil, fmt.Errorf("failed to load JWT_SECRET: %w", err)
	}
	jwtExpiration := 24 * time.Hour

	return &GophermartConfig{
		ServerConfig: ServerConfig{
			RunAddress: runAddr,
			DB:         *dbCfg,
		},
		AccrualSystemAddress: accrualAddr,
		JWTSecret:            jwtSecret,
		JWTExpiration:        jwtExpiration,
	}, nil
}
