package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	// Создаем простой логгер
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()
	log := logger.Sugar()

	// Читаем флаги
	addr := flag.String("a", "localhost:8080", "Server address (e.g. :8080)")
	dsn := flag.String("d", "", "DSN (игнорируется в тестовом сервере)")
	accrualAddr := flag.String("r", "http://localhost:8080", "Accrual system address")

	flag.Parse()

	// Переменные окружения имеют приоритет
	if envAddr := os.Getenv("RUN_ADDRESS"); envAddr != "" {
		*addr = envAddr
	}
	if envDSN := os.Getenv("DATABASE_URI"); envDSN != "" {
		*dsn = envDSN
	}
	if envAccrual := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrual != "" {
		*accrualAddr = envAccrual
	}

	log.Infof("Starting test server on %s", *addr)
	log.Infof("Database DSN: %s (ignored)", *dsn)
	log.Infof("Accrual address: %s", *accrualAddr)

	// Создаем роутер
	router := chi.NewRouter()

	// Health check endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Ping endpoint
	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	// Основные эндпоинты API
	router.Post("/api/user/register", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test-user-id","token":"test-jwt-token"}`))
	})

	router.Post("/api/user/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test-user-id","token":"test-jwt-token"}`))
	})

	// Эндпоинт для тестов с accrual системой
	router.Post("/api/goods", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"match":"Ii0nlklSkq","reward":5,"reward_type":"%"}`))
	})

	// Любые другие эндпоинты возвращают 200 OK
	router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("Received request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"test server response"}`))
	})

	// Создаем сервер
	server := &http.Server{
		Addr:              *addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	// Канал для ошибок сервера
	serverErr := make(chan error, 1)
	go func() {
		log.Infof("Server listening on %s", *addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Проверяем, что сервер запустился
	go func() {
		time.Sleep(100 * time.Millisecond)
		// Пробуем несколько раз подключиться
		for i := 0; i < 10; i++ {
			resp, err := http.Get(fmt.Sprintf("http://%s/health", *addr))
			if err == nil {
				resp.Body.Close()
				log.Info("Server health check passed")
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		log.Warn("Health check failed")
	}()

	// Обработка сигналов завершения
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Ждем сигнал завершения или ошибку сервера
	select {
	case err := <-serverErr:
		log.Fatalf("Server error: %v", err)
	case sig := <-shutdown:
		log.Infof("Received signal: %v. Shutting down...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Errorf("Error during shutdown: %v", err)
		} else {
			log.Info("Server stopped gracefully")
		}
	}
}
