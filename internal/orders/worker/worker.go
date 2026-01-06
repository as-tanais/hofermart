// internal/orders/worker/accrual_worker.go
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

	// Загружаем все награды при старте
	rewards, err := w.rewardRepo.FindAll(ctx)
	if err != nil {
		w.log.Error("Failed to load rewards on startup", zap.Error(err))
	} else {
		w.log.Info("Loaded rewards for processing", zap.Int("count", len(rewards)))
	}

	// Простой цикл обработки
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
	// 1. Получаем заказы со статусом REGISTERED
	orders, err := w.orderRepo.GetOrdersByStatus(ctx, "NEW", 10)
	if err != nil {
		w.log.Error("Failed to get REGISTERED orders", zap.Error(err))
		return
	}

	if len(orders) == 0 {
		// Нет заказов для обработки
		return
	}

	w.log.Debug("Processing orders batch", zap.Int("count", len(orders)))

	// 2. Загружаем текущие награды
	rewards, err := w.rewardRepo.FindAll(ctx)
	if err != nil {
		w.log.Error("Failed to load rewards", zap.Error(err))
		return
	}

	// 3. Обрабатываем каждый заказ
	for _, order := range orders {
		if err := w.processOrder(ctx, order, rewards); err != nil {
			w.log.Error("Failed to process order",
				zap.String("order", order.OrderNumber),
				zap.Error(err))
		}
	}
}

func (w *AccrualWorker) processOrder(ctx context.Context, order orderModel.Order, rewards []rewardModel.Reward) error {
	// 1. Меняем статус на PROCESSING
	if err := w.orderRepo.UpdateOrderStatus(ctx, order.ID, "PROCESSING"); err != nil {
		return err
	}

	w.log.Info("Processing order started",
		zap.String("order", order.OrderNumber),
		zap.String("userID", order.UserID.String()))

	// 2. Получаем товары заказа
	items, err := w.orderRepo.GetOrderItems(ctx, order.ID)
	if err != nil {
		return err
	}

	// 3. Рассчитываем начисления по правилам rewards
	accrual := w.calculateAccrualWithRewards(items, rewards)

	// 4. Определяем финальный статус
	status := "PROCESSED"
	if accrual <= 0 {
		status = "INVALID"
	}

	// 5. Сохраняем результат
	if err := w.orderRepo.UpdateOrderWithAccrual(ctx, order.ID, status, accrual); err != nil {
		return err
	}

	w.log.Info("Order processed",
		zap.String("order", order.OrderNumber),
		zap.String("status", status),
		zap.Float64("accrual", accrual))

	return nil
}

// calculateAccrualWithRewards - рассчитывает начисления по правилам из goods_rewards
func (w *AccrualWorker) calculateAccrualWithRewards(items []orderModel.OrderItem, rewards []rewardModel.Reward) float64 {
	totalAccrual := 0.0

	for _, item := range items {
		itemAccrual := w.calculateItemAccrual(item.Description, item.Price, rewards)
		totalAccrual += itemAccrual
	}

	// Округляем до 2 знаков после запятой
	return roundToTwoDecimals(totalAccrual)
}

// calculateItemAccrual - рассчитывает начисления для одного товара
func (w *AccrualWorker) calculateItemAccrual(description string, price float64, rewards []rewardModel.Reward) float64 {
	itemAccrual := 0.0
	descLower := strings.ToLower(description)

	// Проверяем все правила наград
	for _, reward := range rewards {
		// Проверяем совпадение по паттерну match
		if strings.Contains(descLower, strings.ToLower(reward.Match)) {
			if reward.RewardType == "%" {
				// Процент от цены товара
				accrual := price * (reward.Reward / 100)
				w.log.Debug("Applied percentage reward",
					zap.String("description", description),
					zap.String("match", reward.Match),
					zap.Float64("percentage", reward.Reward),
					zap.Float64("price", price),
					zap.Float64("accrual", accrual))
				itemAccrual += accrual
			} else if reward.RewardType == "pt" {
				// Фиксированные баллы за товар
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
