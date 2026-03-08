package service

import (
	"context"

	"agent-control-plane/apps/api/internal/repo"
)

type DashboardStore interface {
	GetDashboardSummary(ctx context.Context) (repo.DashboardSummary, error)
}

type DashboardService struct {
	store DashboardStore
}

func NewDashboardService(store DashboardStore) *DashboardService {
	return &DashboardService{store: store}
}

func (s *DashboardService) Summary(ctx context.Context) (repo.DashboardSummary, error) {
	if s == nil || s.store == nil {
		return repo.DashboardSummary{}, nil
	}
	return s.store.GetDashboardSummary(ctx)
}
