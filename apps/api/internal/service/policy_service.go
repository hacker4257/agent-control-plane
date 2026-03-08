package service

import (
	"context"

	"agent-control-plane/apps/api/internal/repo"
)

type PolicyStore interface {
	ListPolicyRules(ctx context.Context, enabledOnly bool, limit, offset int) ([]repo.PolicyRule, error)
	GetPolicyRuleByID(ctx context.Context, policyID string) (repo.PolicyRule, error)
	CreatePolicyRule(ctx context.Context, in repo.PolicyRule) (repo.PolicyRule, error)
	UpdatePolicyRule(ctx context.Context, policyID string, in repo.PolicyRule) (repo.PolicyRule, error)
	SetPolicyEnabled(ctx context.Context, policyID string, enabled bool) error
}

type PolicyService struct {
	store PolicyStore
}

func NewPolicyService(store PolicyStore) *PolicyService {
	return &PolicyService{store: store}
}

func (s *PolicyService) List(ctx context.Context, enabledOnly bool, limit, offset int) ([]repo.PolicyRule, error) {
	if s == nil || s.store == nil {
		return []repo.PolicyRule{}, nil
	}
	return s.store.ListPolicyRules(ctx, enabledOnly, limit, offset)
}

func (s *PolicyService) Get(ctx context.Context, policyID string) (repo.PolicyRule, error) {
	if s == nil || s.store == nil {
		return repo.PolicyRule{PolicyID: policyID}, nil
	}
	return s.store.GetPolicyRuleByID(ctx, policyID)
}

func (s *PolicyService) Create(ctx context.Context, in repo.PolicyRule) (repo.PolicyRule, error) {
	if s == nil || s.store == nil {
		return in, nil
	}
	return s.store.CreatePolicyRule(ctx, in)
}

func (s *PolicyService) Update(ctx context.Context, policyID string, in repo.PolicyRule) (repo.PolicyRule, error) {
	if s == nil || s.store == nil {
		in.PolicyID = policyID
		return in, nil
	}
	return s.store.UpdatePolicyRule(ctx, policyID, in)
}

func (s *PolicyService) SetEnabled(ctx context.Context, policyID string, enabled bool) error {
	if s == nil || s.store == nil {
		return nil
	}
	return s.store.SetPolicyEnabled(ctx, policyID, enabled)
}

func (s *PolicyService) EvaluatePreview(ctx context.Context, tool, action, resource, environment string) (string, []string, error) {
	rules, err := s.List(ctx, true, 200, 0)
	if err != nil {
		return "", nil, err
	}
	decision, matched := repo.EvaluatePolicyPreview(rules, tool, action, resource, environment)
	return decision, matched, nil
}
