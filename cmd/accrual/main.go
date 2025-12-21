package main

import (
	"flag"

	"github.com/as-tanais/hofermart/internal/config"
	"github.com/as-tanais/hofermart/internal/logger"
	"go.uber.org/zap"
)

func main() {

	logger := logger.NewLogger()
	defer logger.Sync()

	runAddr := flag.String("a", "localhost", "Server address :8080")
	dbURI := flag.String("d", "postgres", "Database DSN")
	flag.Parse()

	cfg, err := config.LoadAccrualConfig(*runAddr, *dbURI)
	if err != nil {
		logger.Warn("Ошибка", zap.Error(err))
	}

	logger.Info("Загружен конфиг",
		zap.String("run_addr", cfg.RunAddress),
		zap.String("db_uri", cfg.DB.DatabaseURI),
	)
}
