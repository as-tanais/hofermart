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

	// Флаги должны иметь значения по умолчанию
	addr := flag.String("a", "localhost:8080", "Server address (e.g. :8080)")
	dsn := flag.String("d", "postgres://postgres:postgres@localhost:5432/gophermart?sslmode=disable", "DSN")
	accrualAddr := flag.String("r", "http://localhost:8080", "Accrual system address (e.g. http://accrual:8080)")

	flag.Parse()

	// ПРИОРИТЕТ: переменные окружения > флаги
	if envAddr := os.Getenv("RUN_ADDRESS"); envAddr != "" {
		*addr = envAddr
	}
	if envDSN := os.Getenv("DATABASE_URI"); envDSN != "" {
		*dsn = envDSN
	}
	if envAccrual := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrual != "" {
		*accrualAddr = envAccrual
	}

	log.Info("Config loaded",
		zap.String("address", *addr),
		zap.String("dsn", *dsn),
		zap.String("accrual_addr", *accrualAddr))

	cfg, err := config.LoadGophermartConfig(*addr, *dsn, *accrualAddr)
	if err != nil {
		log.Fatal("Не удалось загрузить конфигурацию сервера", zap.Error(err))
	}

	ctx := context.Background()

	log.Info("Connecting to DB...")
	pool, err := postgres.NewPool(ctx, cfg.DB.DatabaseURI)
	if err != nil {
		log.Fatal("DB connection failed", zap.Error(err))
	}
	defer pool.Close()

	log.Info("Starting HTTP server...")

	hasher := hasher.NewHasher(5)
	jwtManager := auth.NewJWTManager("My-strong-sercret-for-JWT-bla-blab-123", 3600)

	userRepo := storage.NewUserStorage(pool)
	userService := service.NewUserService(userRepo, hasher, log)
	userHandler := handler.NewHandler(userService, jwtManager, log)

	router := chi.NewRouter()

	// Добавляем health check для тестов
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Регистрация основных эндпоинтов
	router.Post("/api/user/register", userHandler.Register)
	router.Post("/api/user/login", userHandler.Login)

	// Для тестов - эндпоинт который тесты пытаются использовать
	// Если у вас действительно должен быть POST /api/goods
	// if cfg.AccrualAddress != "" {
	// 	router.Post("/api/goods", func(w http.ResponseWriter, r *http.Request) {
	// 		// Временная заглушка для тестов
	// 		w.WriteHeader(http.StatusOK)
	// 		w.Write([]byte(`{"status":"ok"}`))
	// 	})
	// }

	server := &http.Server{
		Addr:    cfg.RunAddress,
		Handler: router,
		// Важно для быстрого запуска в тестах
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("Server is ready", zap.String("listening on", cfg.RunAddress))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Даем серверу время запуститься
	time.Sleep(100 * time.Millisecond)

	// Проверяем, что сервер запустился
	go func() {
		// Небольшая задержка для стабилизации
		time.Sleep(500 * time.Millisecond)
		resp, err := http.Get("http://" + cfg.RunAddress + "/health")
		if err == nil {
			resp.Body.Close()
			log.Info("Server health check passed")
		} else {
			log.Warn("Health check failed initially", zap.Error(err))
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case errServer := <-serverErr:
		if errServer != nil && errServer != http.ErrServerClosed {
			log.Fatal("Сервер упал", zap.Error(errServer))
		}
	case <-shutdown:
		log.Info("Останавливаем сервер")
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
