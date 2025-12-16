package config

import (
	"os"
	"strconv"
)

func GetEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

func GetEnvIntOrDefault(key string, defaultValue int) (int, error) {
	if val := os.Getenv(key); val != "" {
		return strconv.Atoi(val)
	}
	return defaultValue, nil
}

func GetEnvBoolOrDefault(key string, defaultValue bool) bool {
	if val := os.Getenv(key); val != "" {
		return val == "true" || val == "1"
	}
	return defaultValue
}
