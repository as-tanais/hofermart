package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/as-tanais/hofermart/internal/config"
	"github.com/as-tanais/hofermart/internal/dbmigrate"
	"github.com/as-tanais/hofermart/internal/logger"
	"github.com/as-tanais/hofermart/internal/postgres"
	"github.com/as-tanais/hofermart/internal/server"
	"golang.org/x/sync/errgroup"

	"github.com/as-tanais/hofermart/internal/rewards/handler"
	"github.com/as-tanais/hofermart/internal/rewards/service"
	"github.com/as-tanais/hofermart/internal/rewards/storage"

	accrualHandler "github.com/as-tanais/hofermart/internal/orders/accrual_handler"
	orderSrv "github.com/as-tanais/hofermart/internal/orders/service"
	orderstorage "github.com/as-tanais/hofermart/internal/orders/storage"
	"github.com/as-tanais/hofermart/internal/orders/worker"

	"go.uber.org/zap"
)

func main() {
	log := logger.NewLogger()
	defer log.Sync()

	runAddr := flag.String("a", ":8080", "Server address :8080")
	dbURI := flag.String("d", "postgres://postgres:postgres@localhost:5432/accrual?sslmode=disable", "Database DSN")
	flag.Parse()

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

	// миграции
	log.Info("Applying migrations...")
	if err := dbmigrate.DBMigrate(cfg.DB.DatabaseURI); err != nil {
		log.Fatal("Migration failed", zap.Error(err))
	}
	log.Info("Migrations applied successfully")

	// Подключаемся к БД
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
	orderRepo := orderstorage.NewPostgresStorage(db)

	// Сервис
	rewardService := service.NewService(rewardRepo, log)
	orderService := orderSrv.NewService(orderRepo, log)

	accrualWorker := worker.NewAccrualWorker(orderRepo, rewardRepo, log)

	// Хендлер
	rewardHandler := handler.NewHandler(rewardService, log)
	accrualOrderHandler := accrualHandler.NewAccrualHandler(orderService, log)

	g, gctx := errgroup.WithContext(context.Background())

	g.Go(func() error {
		log.Info("Worker start")
		return accrualWorker.StartWithError(gctx)
	})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/goods", rewardHandler.CreateReward)
	mux.HandleFunc("POST /api/orders", accrualOrderHandler.RegisterOrderWithGoods)
	mux.HandleFunc("GET /api/orders/{number}", accrualOrderHandler.GetOrderStatus)

	serverCfg := server.Config{
		Addr:              cfg.RunAddress,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       30 * time.Second,
		HealthCheckPath:   "/health",
	}

	runner := server.NewRunner(serverCfg, log)
	if err := runner.Run(context.Background()); err != nil {
		log.Fatal("Server failed", zap.Error(err))
	}

}
