package service

import (
	"context"

	rewerr "github.com/as-tanais/hofermart/internal/rewards"
	"github.com/as-tanais/hofermart/internal/rewards/dto"
	"github.com/as-tanais/hofermart/internal/rewards/model"
	"github.com/as-tanais/hofermart/internal/rewards/storage"
	"go.uber.org/zap"
)

type Service struct {
	repo   storage.Repository
	logger *zap.Logger
}

func NewService(repo storage.Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

func (s *Service) validateReward(req *dto.CreateRewardReq) error {
	if req.Match == "" {
		return rewerr.ErrInvalidData
	}
	if req.Reward <= 0 {
		return rewerr.ErrInvalidData
	}
	if req.RewardType != "%" && req.RewardType != "pt" {
		return rewerr.ErrInvalidData
	}
	return nil
}

func (s *Service) CreateReward(req *dto.CreateRewardReq) error {
	if err := s.validateReward(req); err != nil {
		return err
	}

	reward := &model.Reward{
		Match:      req.Match,
		Reward:     req.Reward,
		RewardType: req.RewardType,
	}

	if err := s.repo.Create(context.Background(), reward); err != nil {
		s.logger.Error("Failed to create reward",
			zap.String("match", req.Match),
			zap.Error(err),
		)
		return err
	}

	s.logger.Info("Reward created",
		zap.String("match", req.Match),
		zap.Float64("reward", req.Reward),
		zap.String("type", req.RewardType),
	)
	return nil
}
