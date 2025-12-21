package config

import (
	"fmt"
	"os"
)

func GetEnvOrValue(flagValue, envVarName string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if envValue := os.Getenv(envVarName); envValue != "" {
		return envValue, nil
	}
	return "", fmt.Errorf("missing required config: %s (use flag or %s env var)", envVarName, envVarName)
}
