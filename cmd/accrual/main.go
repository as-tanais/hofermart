package main

import (
	"context"
	"encoding/json"
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
	runAddr := flag.String("a", ":8080", "Server address (e.g. :8080)")
	dbURI := flag.String("d", "postgres://postgres:postgres@localhost:5432/accrual?sslmode=disable", "Database DSN")
	flag.Parse()

	// Переменные окружения имеют приоритет
	if envAddr := os.Getenv("RUN_ADDRESS"); envAddr != "" {
		*runAddr = envAddr
	}
	if envDBURI := os.Getenv("DATABASE_URI"); envDBURI != "" {
		*dbURI = envDBURI
	}

	log.Infof("Starting test accrual server on %s", *runAddr)
	log.Infof("Database DSN: %s (ignored)", *dbURI)

	// Хранилище для тестовых данных
	rewards := []map[string]interface{}{
		{"match": "Ii0nlklSkq", "reward": 5, "reward_type": "%"},
		{"match": "test", "reward": 10, "reward_type": "pt"},
	}

	orders := make(map[string]map[string]interface{})

	// Создаем роутер
	router := chi.NewRouter()

	// Health check endpoint
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Ping endpoint
	router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	// API для goods (наград)
	router.Post("/api/goods", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("POST /api/goods")

		var newReward map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&newReward); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
			return
		}

		// Добавляем ID
		newReward["id"] = len(rewards) + 1
		rewards = append(rewards, newReward)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(newReward)
	})

	router.Get("/api/goods", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("GET /api/goods")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(rewards)
	})

	// API для orders
	router.Post("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("POST /api/orders")

		var order map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
			return
		}

		orderNumber, ok := order["order"].(string)
		if !ok || orderNumber == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "order number is required"})
			return
		}

		// Сохраняем заказ
		orders[orderNumber] = map[string]interface{}{
			"order":   orderNumber,
			"status":  "PROCESSING",
			"accrual": 0,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"order":  orderNumber,
			"status": "REGISTERED",
		})
	})

	router.Get("/api/orders/{number}", func(w http.ResponseWriter, r *http.Request) {
		number := chi.URLParam(r, "number")
		log.Infof("GET /api/orders/%s", number)

		order, exists := orders[number]
		if !exists {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Симулируем обработку заказа
		if order["status"] == "PROCESSING" {
			// Через некоторое время меняем статус
			orders[number] = map[string]interface{}{
				"order":   number,
				"status":  "PROCESSED",
				"accrual": 100.5,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(orders[number])
	})

	// Эндпоинт для тестов - возвращает статус начисления
	router.Get("/api/accrual/{number}", func(w http.ResponseWriter, r *http.Request) {
		number := chi.URLParam(r, "number")
		log.Infof("GET /api/accrual/%s", number)

		// Симулируем разные статусы
		statuses := []string{"REGISTERED", "PROCESSING", "PROCESSED", "INVALID"}
		statusIndex := len(number) % len(statuses)
		status := statuses[statusIndex]

		response := map[string]interface{}{
			"order":  number,
			"status": status,
		}

		if status == "PROCESSED" {
			response["accrual"] = 500.0
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	// Эндпоинт для регистрации магазинов/товаров
	router.Post("/api/register", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("POST /api/register")

		var data map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":      time.Now().Unix(),
			"success": true,
			"data":    data,
		})
	})

	// Дефолтный обработчик для всех остальных запросов
	router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		log.Infof("%s %s", r.Method, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"service": "test-accrual",
			"status":  "running",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// Создаем сервер
	server := &http.Server{
		Addr:              *runAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	// Канал для ошибок сервера
	serverErr := make(chan error, 1)
	go func() {
		log.Infof("Accrual server listening on %s", *runAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Проверяем, что сервер запустился
	go func() {
		time.Sleep(100 * time.Millisecond)
		// Пробуем несколько раз подключиться
		for i := 0; i < 10; i++ {
			resp, err := http.Get(fmt.Sprintf("http://%s/health", *runAddr))
			if err == nil {
				resp.Body.Close()
				log.Info("Accrual server health check passed")
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		log.Warn("Accrual server health check failed")
	}()

	// Обработка сигналов завершения
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Ждем сигнал завершения или ошибку сервера
	select {
	case err := <-serverErr:
		log.Fatalf("Accrual server error: %v", err)
	case sig := <-shutdown:
		log.Infof("Received signal: %v. Shutting down accrual server...", sig)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Errorf("Error during accrual server shutdown: %v", err)
		} else {
			log.Info("Accrual server stopped gracefully")
		}
	}
}
