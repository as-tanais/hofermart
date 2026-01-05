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
	"github.com/as-tanais/hofermart/internal/middleware"
	"github.com/as-tanais/hofermart/internal/postgres"

	// User
	userHandler "github.com/as-tanais/hofermart/internal/user/handler"
	userService "github.com/as-tanais/hofermart/internal/user/service"
	userStorage "github.com/as-tanais/hofermart/internal/user/storage"

	// Order (gophermart)
	orderHandler "github.com/as-tanais/hofermart/internal/orders/handler"
	orderService "github.com/as-tanais/hofermart/internal/orders/service"
	orderStorage "github.com/as-tanais/hofermart/internal/orders/storage"

	balanceHandler "github.com/as-tanais/hofermart/internal/balance/handler"
	balanceService "github.com/as-tanais/hofermart/internal/balance/service"
	balanceStorage "github.com/as-tanais/hofermart/internal/balance/storage"

	"github.com/as-tanais/hofermart/internal/utils/hasher"
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

	log.Info("Applying migrations...")
	if err := dbmigrate.DBMigrate(cfg.DB.DatabaseURI); err != nil {
		log.Fatal("Migration failed", zap.Error(err))
	}

	ctx := context.Background()

	log.Info("Connecting to DB...")
	pool, err := postgres.NewPool(ctx, cfg.DB.DatabaseURI)
	if err != nil {
		log.Fatal("DB connection failed", zap.Error(err))
	}
	defer pool.Close()

	log.Info("Starting HTTP server...")

	// Инициализация зависимостей
	hasher := hasher.NewHasher(5)
	jwtManager := auth.NewJWTManager("My-strong-secret-for-JWT-bla-blab-123", 3600*time.Second)

	// User сервисы
	userRepo := userStorage.NewUserStorage(pool)
	userSvc := userService.NewUserService(userRepo, hasher, log)
	userHdl := userHandler.NewHandler(userSvc, jwtManager, log)

	// Order сервисы (gophermart)
	orderRepo := orderStorage.NewPostgresStorage(pool)
	orderSvc := orderService.NewService(orderRepo, log)
	orderHdl := orderHandler.NewHandler(orderSvc, log)

	// Balance
	balanceRepo := balanceStorage.NewPostgresStorage(pool)
	balanceSvc := balanceService.NewBalanceService(balanceRepo, log)
	balanceHdl := balanceHandler.NewBalanceHandler(balanceSvc, log)

	// Создаем роутер
	router := chi.NewRouter()

	router.Group(func(r chi.Router) {
		// Health checks
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})

		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("pong"))
		})

		// Регистрация и логин
		r.Post("/api/user/register", userHdl.Register)
		r.Post("/api/user/login", userHdl.Login)
	})

	// ========== PROTECTED ROUTES (требуют аутентификации) ==========
	router.Group(func(r chi.Router) {
		// Применяем middleware проверки аутентификации
		r.Use(middleware.AuthMiddleware(jwtManager, log))

		// Заказы пользователя
		r.Post("/api/user/orders", orderHdl.RegisterOrder)
		r.Get("/api/user/orders", orderHdl.GetUserOrders)

		// Баланс
		r.Get("/api/user/balance", balanceHdl.GetBalance)

		r.Post("/api/user/balance/withdraw", balanceHdl.Withdraw)
		r.Get("/api/user/withdrawals", balanceHdl.GetWithdrawals)

		// Логаут (удаляет куку)
		r.Post("/api/user/logout", func(w http.ResponseWriter, r *http.Request) {
			middleware.ClearAuthCookie(w)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"logged out"}`))
		})
	})

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
			log.Error("Server error", zap.Error(err))
			serverErr <- err
		}
	}()

	// Проверка запуска
	go func() {
		time.Sleep(500 * time.Millisecond)

		client := &http.Client{Timeout: 2 * time.Second}
		url := "http://" + cfg.RunAddress + "/health"

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

	// Graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

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
