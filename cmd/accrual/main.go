package main

func main() {}

// import (
// 	"context"
// 	"flag"
// 	"net/http"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"github.com/as-tanais/hofermart/internal/config"
// 	"github.com/as-tanais/hofermart/internal/logger"
// 	"github.com/go-chi/chi/v5"
// 	"go.uber.org/zap"
// )

// func main() {

// 	log := logger.NewLogger()
// 	defer log.Sync()

// 	runAddr := flag.String("a", ":8080", "Server address :8080")
// 	dbURI := flag.String("d", "postgres", "Database DSN")
// 	flag.Parse()

// 	cfg, err := config.LoadAccrualConfig(*runAddr, *dbURI)
// 	if err != nil {
// 		log.Warn("Ошибка", zap.Error(err))
// 	}

// 	log.Info("Applying migrations...")

// 	// if err := dbmigrate.DBMigrate(cfg.DB.DatabaseURI); err != nil {
// 	// 	log.Fatal("Migration failed", zap.Error(err))
// 	// }

// 	// log.Info("Connecting to DB...")
// 	// ctx := context.Background()

// 	// db, err := postgres.NewPool(ctx, cfg.DB.DatabaseURI)
// 	// if err != nil {
// 	// 	log.Fatal("DB connection failed", zap.Error(err))
// 	// }
// 	// defer db.Close()

// 	// Репозиторий
// 	// rewardRepo := storage.NewPostgresStorage(db)

// 	// // Сервис
// 	// rewardService := service.NewService(rewardRepo, log)

// 	// // Хендлер
// 	// rewardHandler := handler.NewHandler(rewardService, log)

// 	// // Репозиторий заказов
// 	// orderRepo := orderstorage.NewPostgresStorage(db)

// 	// // Сервис заказов
// 	// orderService := orderSrv.NewService(orderRepo, log)

// 	// // Хендлер
// 	// orderHandler := orderHand.NewHandler(orderService, log)

// 	// Роутер
// 	router := chi.NewRouter()
// 	// router.Post("/api/goods", rewardHandler.CreateReward)
// 	// router.Post("/api/orders", orderHandler.RegisterOrder)
// 	// router.Get("/api/orders/{number}", orderHandler.GetOrder)

// 	// Запуск сервера
// 	server := &http.Server{
// 		Addr:    cfg.RunAddress,
// 		Handler: router,
// 	}

// 	serverErr := make(chan error, 1)
// 	go func() {
// 		log.Info("Server is ready", zap.String("listening on", cfg.RunAddress))
// 		serverErr <- server.ListenAndServe()
// 	}()

// 	shutdown := make(chan os.Signal, 1)
// 	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

// 	var errServer error
// 	select {
// 	case errServer = <-serverErr:
// 		if errServer != nil && errServer != http.ErrServerClosed {
// 			log.Fatal("Сервер упал", zap.Error(errServer))
// 		}
// 	case <-shutdown:
// 		log.Info("Останавлваем сервере")
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	if err := server.Shutdown(ctx); err != nil {
// 		log.Error("Failed to gracefully shutdown server", zap.Error(err))
// 		os.Exit(1)
// 	} else {
// 		log.Info("Server stopped")
// 	}

// }
