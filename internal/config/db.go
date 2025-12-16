package config

import (
	"flag"
	"fmt"
)

type DBConfig struct {
	DATABASE_URI string
}

func getDBURI() (string, error) {

	dbFlag := flag.String("d", "", "Database dsn")
	flag.Parse()
	dbUri := GetEnvOrDefault("DATABASE_URI", *dbFlag)

	if dbUri == "" {
		return "", fmt.Errorf("отсутсвует адрес подключения к БД")
	}

	return dbUri, nil
}
