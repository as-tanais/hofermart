package worker

import (
	"context"
	"strings"
	"time"

	orderModel "github.com/as-tanais/hofermart/internal/orders/model"
	"github.com/as-tanais/hofermart/internal/orders/storage"
	rewardModel "github.com/as-tanais/hofermart/internal/rewards/model"
	rewardStorage "github.com/as-tanais/hofermart/internal/rewards/storage"
	"go.uber.org/zap"
)

type AccrualWorker struct {
	orderRepo  storage.Repository
	rewardRepo rewardStorage.Repository
	log        *zap.Logger
}

func NewAccrualWorker(
	orderRepo storage.Repository,
	rewardRepo rewardStorage.Repository,
	log *zap.Logger,
) *AccrualWorker {
	return &AccrualWorker{
		orderRepo:  orderRepo,
		rewardRepo: rewardRepo,
		log:        log,
	}
}

func (w *AccrualWorker) Start(ctx context.Context) {
	w.log.Info("Starting accrual worker")

	rewards, err := w.rewardRepo.FindAll(ctx)
	if err != nil {
		w.log.Error("Failed to load rewards on startup", zap.Error(err))
	} else {
		w.log.Info("Loaded rewards for processing", zap.Int("count", len(rewards)))
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("Accrual worker stopped")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *AccrualWorker) processBatch(ctx context.Context) {

	orders, err := w.orderRepo.GetOrdersByStatus(ctx, "NEW", 10)
	if err != nil {
		w.log.Error("Failed to get NEW orders", zap.Error(err))
		return
	}

	if len(orders) == 0 {

		return
	}

	w.log.Debug("Processing orders batch", zap.Int("count", len(orders)))

	rewards, err := w.rewardRepo.FindAll(ctx)
	if err != nil {
		w.log.Error("Failed to load rewards", zap.Error(err))
		return
	}

	for _, order := range orders {
		if err := w.processOrder(ctx, order, rewards); err != nil {
			w.log.Error("Failed to process order",
				zap.String("order", order.OrderNumber),
				zap.Error(err))
		}
	}
}

func (w *AccrualWorker) processOrder(ctx context.Context, order orderModel.Order, rewards []rewardModel.Reward) error {

	if err := w.orderRepo.UpdateOrderStatus(ctx, order.ID, "PROCESSING"); err != nil {
		return err
	}

	w.log.Info("Processing order started",
		zap.String("order", order.OrderNumber),
		zap.String("userID", order.UserID.String()))

	items, err := w.orderRepo.GetOrderItems(ctx, order.ID)
	if err != nil {
		return err
	}

	accrual := w.calculateAccrualWithRewards(items, rewards)

	status := "PROCESSED"
	if accrual <= 0 {
		status = "INVALID"
	}

	if err := w.orderRepo.UpdateOrderWithAccrual(ctx, order.ID, status, accrual); err != nil {
		return err
	}

	w.log.Info("Order processed",
		zap.String("order", order.OrderNumber),
		zap.String("status", status),
		zap.Float64("accrual", accrual))

	return nil
}

func (w *AccrualWorker) calculateAccrualWithRewards(items []orderModel.OrderItem, rewards []rewardModel.Reward) float64 {
	totalAccrual := 0.0

	for _, item := range items {
		itemAccrual := w.calculateItemAccrual(item.Description, item.Price, rewards)
		totalAccrual += itemAccrual
	}

	return roundToTwoDecimals(totalAccrual)
}

func (w *AccrualWorker) calculateItemAccrual(description string, price float64, rewards []rewardModel.Reward) float64 {
	itemAccrual := 0.0
	descLower := strings.ToLower(description)

	for _, reward := range rewards {

		if strings.Contains(descLower, strings.ToLower(reward.Match)) {
			if reward.RewardType == "%" {

				accrual := price * (reward.Reward / 100)
				w.log.Debug("Applied percentage reward",
					zap.String("description", description),
					zap.String("match", reward.Match),
					zap.Float64("percentage", reward.Reward),
					zap.Float64("price", price),
					zap.Float64("accrual", accrual))
				itemAccrual += accrual
			} else if reward.RewardType == "pt" {

				w.log.Debug("Applied points reward",
					zap.String("description", description),
					zap.String("match", reward.Match),
					zap.Float64("points", reward.Reward))
				itemAccrual += reward.Reward
			}
		}
	}

	return itemAccrual
}

func roundToTwoDecimals(value float64) float64 {
	return float64(int(value*100+0.5)) / 100
}
