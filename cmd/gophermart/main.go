package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/as-tanais/hofermart/internal/auth"
	"github.com/as-tanais/hofermart/internal/config"
	"github.com/as-tanais/hofermart/internal/dbmigrate"
	"github.com/as-tanais/hofermart/internal/logger"
	"github.com/as-tanais/hofermart/internal/postgres"
	"github.com/as-tanais/hofermart/internal/user/handler"
	"github.com/as-tanais/hofermart/internal/user/service"
	"github.com/as-tanais/hofermart/internal/user/storage"
	"github.com/as-tanais/hofermart/internal/utils/hasher"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {

	log := logger.NewLogger()

	addr := flag.String("a", ":8080", "Server address (e.g. :8080)")
	dsn := flag.String("d", "", "DSN")
	accrualAddr := flag.String("r", "", "Accrual system address (e.g. http://accrual:8080)")

	flag.Parse()

	cfg, err := config.LoadGophermartConfig(*addr, *dsn, *accrualAddr)
	if err != nil {
		log.Fatal("Не удалось загрузить конфигурацию сервера", zap.Error(err))
	}

	if err := dbmigrate.DBMigrate(cfg.DB.DatabaseURI); err != nil {
		log.Fatal("Migration failed", zap.Error(err))
	}

	ctx := context.Background()

	pool, err := postgres.NewPool(ctx, cfg.DB.DatabaseURI)
	if err != nil {
		log.Fatal("DB connection failed", zap.Error(err))
	}

	hasher := hasher.NewHasher(5)
	jwtManager := auth.NewJWTManager("My-strong-sercret-for-JWT-bla-blab-123", 3600)

	userRepo := storage.NewUserStorage(pool)
	userService := service.NewUserService(userRepo, hasher, log)
	userHandler := handler.NewHandler(userService, jwtManager, log)

	router := chi.NewRouter()
	server := &http.Server{Addr: cfg.RunAddress, Handler: router}

	router.Post("/api/user/register", userHandler.Register)
	router.Post("/api/user/login", userHandler.Login)

	serverErr := make(chan error, 1)
	go func() {
		log.Info("Server is ready", zap.String("listening on", cfg.RunAddress))
		serverErr <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	var errServer error
	select {
	case errServer = <-serverErr:
		if errServer != nil && errServer != http.ErrServerClosed {
			log.Fatal("Сервер упал", zap.Error(errServer))
		}
	case <-shutdown:
		log.Info("Останавлваем сервере")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("Failed to gracefully shutdown server", zap.Error(err))
		os.Exit(1)
	} else {
		log.Info("Server stopped")
	}

}
