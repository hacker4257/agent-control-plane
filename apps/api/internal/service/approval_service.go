package service

import (
	"context"

	"agent-control-plane/apps/api/internal/repo"
)

type ApprovalStore interface {
	ListApprovals(ctx context.Context, status string, limit, offset int) ([]repo.ApprovalRecord, error)
	GetApprovalByID(ctx context.Context, approvalID string) (repo.ApprovalRecord, error)
	ApplyApprovalDecision(ctx context.Context, in repo.ApprovalDecisionInput) (repo.ApprovalRecord, error)
}

type ApprovalService struct {
	store ApprovalStore
}

func NewApprovalService(store ApprovalStore) *ApprovalService {
	return &ApprovalService{store: store}
}

func (s *ApprovalService) List(ctx context.Context, status string, limit, offset int) ([]repo.ApprovalRecord, error) {
	if s == nil || s.store == nil {
		return []repo.ApprovalRecord{}, nil
	}
	return s.store.ListApprovals(ctx, status, limit, offset)
}

func (s *ApprovalService) Get(ctx context.Context, approvalID string) (repo.ApprovalRecord, error) {
	if s == nil || s.store == nil {
		return repo.ApprovalRecord{ApprovalID: approvalID}, nil
	}
	return s.store.GetApprovalByID(ctx, approvalID)
}

func (s *ApprovalService) Decide(ctx context.Context, in repo.ApprovalDecisionInput) (repo.ApprovalRecord, error) {
	if s == nil || s.store == nil {
		return repo.ApprovalRecord{ApprovalID: in.ApprovalID, Status: "approved"}, nil
	}
	return s.store.ApplyApprovalDecision(ctx, in)
}
