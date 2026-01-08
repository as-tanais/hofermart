package logger

import "go.uber.org/zap"

func NewLogger() *zap.Logger {
	logger, err := zap.NewDevelopment()

	if err != nil {
		panic("не удалось создать логгер")
	}

	return logger
}
