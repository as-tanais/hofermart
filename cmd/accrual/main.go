package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/as-tanais/hofermart/internal/config"
	"github.com/as-tanais/hofermart/internal/dbmigrate"
	"github.com/as-tanais/hofermart/internal/logger"
	"github.com/as-tanais/hofermart/internal/postgres"

	"github.com/as-tanais/hofermart/internal/rewards/handler"
	"github.com/as-tanais/hofermart/internal/rewards/service"
	"github.com/as-tanais/hofermart/internal/rewards/storage"

	orderHand "github.com/as-tanais/hofermart/internal/orders/handler"
	orderSrv "github.com/as-tanais/hofermart/internal/orders/service"
	orderstorage "github.com/as-tanais/hofermart/internal/orders/storage"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	log := logger.NewLogger()
	defer log.Sync()

	runAddr := flag.String("a", ":8080", "Server address :8080")
	dbURI := flag.String("d", "postgres://postgres:postgres@localhost:5432/accrual?sslmode=disable", "Database DSN")
	flag.Parse()

	// ПРИОРИТЕТ: переменные окружения > флаги
	if envAddr := os.Getenv("RUN_ADDRESS"); envAddr != "" {
		*runAddr = envAddr
	}
	if envDBURI := os.Getenv("DATABASE_URI"); envDBURI != "" {
		*dbURI = envDBURI
	}

	log.Info("Config loaded",
		zap.String("address", *runAddr),
		zap.String("database", *dbURI))

	cfg, err := config.LoadAccrualConfig(*runAddr, *dbURI)
	if err != nil {
		log.Fatal("Failed to load config", zap.Error(err))
	}

	// 1. Выполняем миграции
	log.Info("Applying migrations...")
	if err := dbmigrate.DBMigrate(cfg.DB.DatabaseURI); err != nil {
		log.Fatal("Migration failed", zap.Error(err))
	}
	log.Info("Migrations applied successfully")

	// 2. Подключаемся к БД
	log.Info("Connecting to DB...")
	ctx := context.Background()

	db, err := postgres.NewPool(ctx, cfg.DB.DatabaseURI)
	if err != nil {
		log.Fatal("DB connection failed", zap.Error(err))
	}
	defer db.Close()

	// Проверяем подключение
	if err := db.Ping(ctx); err != nil {
		log.Fatal("DB ping failed", zap.Error(err))
	}
	log.Info("DB connection established")

	// Репозиторий
	rewardRepo := storage.NewPostgresStorage(db)

	// Сервис
	rewardService := service.NewService(rewardRepo, log)

	// Хендлер
	rewardHandler := handler.NewHandler(rewardService, log)

	// Репозиторий заказов
	orderRepo := orderstorage.NewPostgresStorage(db)

	// Сервис заказов
	orderService := orderSrv.NewService(orderRepo, log)

	// Хендлер
	orderHandler := orderHand.NewHandler(orderService, log)
	// Роутер
	router := chi.NewRouter()

	// Health check с проверкой БД
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		// Проверяем подключение к БД
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"db_error"}`))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Основные эндпоинты
	router.Post("/api/goods", rewardHandler.CreateReward)
	router.Post("/api/orders", orderHandler.RegisterOrder)
	router.Get("/api/orders/{number}", orderHandler.GetOrder)

	// Запуск сервера
	server := &http.Server{
		Addr:              cfg.RunAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("Server is ready", zap.String("listening on", cfg.RunAddress))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Даем серверу время запуститься и проверяем health
	go func() {
		time.Sleep(500 * time.Millisecond)
		client := &http.Client{Timeout: 2 * time.Second}
		url := "http://" + cfg.RunAddress + "/health"
		if cfg.RunAddress[0] == ':' {
			url = "http://localhost" + cfg.RunAddress + "/health"
		}
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			log.Info("Server health check passed")
		} else {
			log.Warn("Health check failed", zap.Error(err))
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	var errServer error
	select {
	case errServer = <-serverErr:
		if errServer != nil && errServer != http.ErrServerClosed {
			log.Fatal("Server crashed", zap.Error(errServer))
		}
	case <-shutdown:
		log.Info("Shutting down server...")
	}

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Error("Failed to gracefully shutdown server", zap.Error(err))
		os.Exit(1)
	} else {
		log.Info("Server stopped")
	}
}
