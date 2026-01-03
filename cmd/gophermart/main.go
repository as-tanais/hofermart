package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	// "github.com/as-tanais/hofermart/internal/auth"
	"github.com/as-tanais/hofermart/internal/config"
	"github.com/as-tanais/hofermart/internal/logger"

	// Пока закомментируем БД и миграции
	// "github.com/as-tanais/hofermart/internal/postgres"
	// "github.com/as-tanais/hofermart/internal/user/handler"
	// "github.com/as-tanais/hofermart/internal/user/service"
	// "github.com/as-tanais/hofermart/internal/user/storage"
	// "github.com/as-tanais/hofermart/internal/utils/hasher"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	// Создаем логгер с подробным выводом для отладки
	log := logger.NewLogger()
	defer log.Sync()

	// Устанавливаем значения по умолчанию для флагов
	addr := flag.String("a", "localhost:8080", "Server address (e.g. :8080)")
	dsn := flag.String("d", "postgres://postgres:postgres@localhost:5432/gophermart?sslmode=disable", "DSN")
	accrualAddr := flag.String("r", "http://localhost:8080", "Accrual system address (e.g. http://accrual:8080)")

	flag.Parse()

	// ПРИОРИТЕТ: переменные окружения > флаги
	if envAddr := os.Getenv("RUN_ADDRESS"); envAddr != "" {
		*addr = envAddr
		log.Info("Using RUN_ADDRESS from environment", zap.String("value", envAddr))
	}
	if envDSN := os.Getenv("DATABASE_URI"); envDSN != "" {
		*dsn = envDSN
		log.Info("Using DATABASE_URI from environment", zap.String("value", envDSN))
	}
	if envAccrual := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrual != "" {
		*accrualAddr = envAccrual
		log.Info("Using ACCRUAL_SYSTEM_ADDRESS from environment", zap.String("value", envAccrual))
	}

	log.Info("Config loaded",
		zap.String("address", *addr),
		zap.String("dsn", *dsn),
		zap.String("accrual_addr", *accrualAddr))

	cfg, err := config.LoadGophermartConfig(*addr, *dsn, *accrualAddr)
	if err != nil {
		log.Fatal("Failed to load config", zap.Error(err))
	}

	// TODO: Раскомментировать когда понадобится
	// log.Info("Applying migrations...")
	// if err := dbmigrate.DBMigrate(cfg.DB.DatabaseURI); err != nil {
	// 	log.Fatal("Migration failed", zap.Error(err))
	// }

	// ctx := context.Background()

	// TODO: Раскомментировать когда понадобится БД
	// log.Info("Connecting to DB...")
	// pool, err := postgres.NewPool(ctx, cfg.DB.DatabaseURI)
	// if err != nil {
	// 	log.Fatal("DB connection failed", zap.Error(err))
	// }
	// defer pool.Close()

	log.Info("Starting HTTP server...")

	// TODO: Раскомментировать когда понадобится
	// hasher := hasher.NewHasher(5)
	// jwtManager := auth.NewJWTManager("My-strong-secret-for-JWT-bla-blab-123", 3600)
	//
	// userRepo := storage.NewUserStorage(pool)
	// userService := service.NewUserService(userRepo, hasher, log)
	// userHandler := handler.NewHandler(userService, jwtManager, log)

	router := chi.NewRouter()

	// Обязательно добавьте health check для тестов!
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Добавим ping для простой проверки
	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	// TODO: Раскомментировать когда хендлеры будут готовы
	// router.Post("/api/user/register", userHandler.Register)
	// router.Post("/api/user/login", userHandler.Login)

	// Временные заглушки для тестов
	router.Post("/api/user/register", func(w http.ResponseWriter, r *http.Request) {
		log.Info("POST /api/user/register called")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test-user-id","token":"test-jwt-token"}`))
	})

	router.Post("/api/user/login", func(w http.ResponseWriter, r *http.Request) {
		log.Info("POST /api/user/login called")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test-user-id","token":"test-jwt-token"}`))
	})

	// Эндпоинт для тестов с accrual системой
	router.Post("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		log.Info("POST /api/orders called")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"order":"123456","status":"REGISTERED"}`))
	})

	router.Get("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		log.Info("GET /api/orders called")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})

	router.Get("/api/user/balance", func(w http.ResponseWriter, r *http.Request) {
		log.Info("GET /api/user/balance called")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"current":500.5,"withdrawn":100.0}`))
	})

	router.Post("/api/user/balance/withdraw", func(w http.ResponseWriter, r *http.Request) {
		log.Info("POST /api/user/balance/withdraw called")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"order":"123456","sum":100.0}`))
	})

	router.Get("/api/user/withdrawals", func(w http.ResponseWriter, r *http.Request) {
		log.Info("GET /api/user/withdrawals called")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})

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
			log.Error("Server error", zap.Error(err))
			serverErr <- err
		}
	}()

	// Проверяем что сервер действительно запустился
	go func() {
		// Даем серверу время на запуск
		time.Sleep(500 * time.Millisecond)

		// Проверяем health endpoint
		client := &http.Client{Timeout: 2 * time.Second}
		url := "http://" + cfg.RunAddress + "/health"
		// Если адрес начинается с двоеточия, добавляем localhost
		if cfg.RunAddress[0] == ':' {
			url = "http://localhost" + cfg.RunAddress + "/health"
		}

		resp, err := client.Get(url)
		if err != nil {
			log.Warn("Health check failed (server might still be starting)",
				zap.Error(err),
				zap.String("url", url))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.Info("Server health check passed")
		} else {
			log.Warn("Health check returned non-200 status",
				zap.Int("status", resp.StatusCode))
		}
	}()

	// Ожидаем OS сигналов для graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	// Ждем либо ошибки сервера, либо сигнала завершения
	select {
	case errServer := <-serverErr:
		if errServer != nil && errServer != http.ErrServerClosed {
			log.Fatal("Server crashed", zap.Error(errServer))
		}
		log.Info("Server stopped normally")
	case sig := <-shutdown:
		log.Info("Received shutdown signal", zap.String("signal", sig.String()))
	}

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("Failed to gracefully shutdown server", zap.Error(err))
		os.Exit(1)
	} else {
		log.Info("Server stopped gracefully")
	}
}
