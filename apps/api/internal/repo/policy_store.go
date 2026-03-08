package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type PolicyRule struct {
	PolicyID         string                 `json:"policy_id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	ScopeAgent       string                 `json:"scope_agent"`
	ScopeTool        string                 `json:"scope_tool"`
	ScopeEnvironment string                 `json:"scope_environment"`
	ScopeResourcePat string                 `json:"scope_resource_pat"`
	ConditionExpr    map[string]interface{} `json:"condition_expr"`
	Decision         string                 `json:"decision"`
	Priority         int                    `json:"priority"`
	Enabled          bool                   `json:"enabled"`
	CreatedBy        string                 `json:"created_by"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type policyRowsIface interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
	Close()
}

type policyRowIface interface {
	Scan(dest ...interface{}) error
}

type policyExecTagIface interface {
	RowsAffected() int64
}

type policyQueryPool interface {
	Query(ctx context.Context, sql string, args ...interface{}) (policyRowsIface, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) policyRowIface
	Exec(ctx context.Context, sql string, args ...interface{}) (policyExecTagIface, error)
}

type storePolicyPoolAdapter struct {
	store *Store
}

func (a *storePolicyPoolAdapter) Query(ctx context.Context, sql string, args ...interface{}) (policyRowsIface, error) {
	return a.store.pool.Query(ctx, sql, args...)
}

func (a *storePolicyPoolAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) policyRowIface {
	return a.store.pool.QueryRow(ctx, sql, args...)
}

func (a *storePolicyPoolAdapter) Exec(ctx context.Context, sql string, args ...interface{}) (policyExecTagIface, error) {
	return a.store.pool.Exec(ctx, sql, args...)
}

func (s *Store) ListPolicyRules(ctx context.Context, enabledOnly bool, limit, offset int) ([]PolicyRule, error) {
	return listPolicyRulesWithPool(ctx, &storePolicyPoolAdapter{store: s}, enabledOnly, limit, offset)
}

func (s *Store) GetPolicyRuleByID(ctx context.Context, policyID string) (PolicyRule, error) {
	return getPolicyRuleByIDWithPool(ctx, &storePolicyPoolAdapter{store: s}, policyID)
}

func (s *Store) CreatePolicyRule(ctx context.Context, in PolicyRule) (PolicyRule, error) {
	if in.PolicyID == "" {
		in.PolicyID = "pol_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	}
	if in.Priority == 0 {
		in.Priority = 100
	}
	if in.ConditionExpr == nil {
		in.ConditionExpr = map[string]interface{}{}
	}

	cond, err := json.Marshal(in.ConditionExpr)
	if err != nil {
		return PolicyRule{}, err
	}

	var out PolicyRule
	var condBytes []byte
	err = s.pool.QueryRow(ctx, `
insert into policy_rules (
  policy_id,
  name,
  description,
  scope_agent,
  scope_tool,
  scope_environment,
  scope_resource_pat,
  condition_expr,
  decision,
  priority,
  enabled,
  created_by
) values (
  $1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9,$10,$11,$12
)
returning
  policy_id,
  name,
  coalesce(description, ''),
  coalesce(scope_agent, ''),
  coalesce(scope_tool, ''),
  coalesce(scope_environment, ''),
  coalesce(scope_resource_pat, ''),
  condition_expr,
  decision,
  priority,
  enabled,
  coalesce(created_by, ''),
  created_at,
  updated_at
`,
		in.PolicyID,
		in.Name,
		nullIfEmpty(in.Description),
		nullIfEmpty(in.ScopeAgent),
		nullIfEmpty(in.ScopeTool),
		nullIfEmpty(in.ScopeEnvironment),
		nullIfEmpty(in.ScopeResourcePat),
		string(cond),
		in.Decision,
		in.Priority,
		in.Enabled,
		nullIfEmpty(in.CreatedBy),
	).Scan(
		&out.PolicyID,
		&out.Name,
		&out.Description,
		&out.ScopeAgent,
		&out.ScopeTool,
		&out.ScopeEnvironment,
		&out.ScopeResourcePat,
		&condBytes,
		&out.Decision,
		&out.Priority,
		&out.Enabled,
		&out.CreatedBy,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return PolicyRule{}, err
	}
	if err := json.Unmarshal(condBytes, &out.ConditionExpr); err != nil {
		return PolicyRule{}, err
	}
	return out, nil
}

func (s *Store) UpdatePolicyRule(ctx context.Context, policyID string, in PolicyRule) (PolicyRule, error) {
	cond := in.ConditionExpr
	if cond == nil {
		cond = map[string]interface{}{}
	}
	condJSON, err := json.Marshal(cond)
	if err != nil {
		return PolicyRule{}, err
	}

	tag, err := s.pool.Exec(ctx, `
update policy_rules
set
  name = $2,
  description = $3,
  scope_agent = $4,
  scope_tool = $5,
  scope_environment = $6,
  scope_resource_pat = $7,
  condition_expr = $8::jsonb,
  decision = $9,
  priority = $10,
  updated_at = now()
where policy_id = $1
`,
		policyID,
		in.Name,
		nullIfEmpty(in.Description),
		nullIfEmpty(in.ScopeAgent),
		nullIfEmpty(in.ScopeTool),
		nullIfEmpty(in.ScopeEnvironment),
		nullIfEmpty(in.ScopeResourcePat),
		string(condJSON),
		in.Decision,
		in.Priority,
	)
	if err != nil {
		return PolicyRule{}, err
	}
	if tag.RowsAffected() == 0 {
		return PolicyRule{}, ErrNotFound
	}
	return s.GetPolicyRuleByID(ctx, policyID)
}

func (s *Store) SetPolicyEnabled(ctx context.Context, policyID string, enabled bool) error {
	return setPolicyEnabledWithPool(ctx, &storePolicyPoolAdapter{store: s}, policyID, enabled)
}

func listPolicyRulesWithPool(ctx context.Context, pool policyQueryPool, enabledOnly bool, limit, offset int) ([]PolicyRule, error) {
	rows, err := pool.Query(ctx, `
select
  policy_id,
  name,
  coalesce(description, ''),
  coalesce(scope_agent, ''),
  coalesce(scope_tool, ''),
  coalesce(scope_environment, ''),
  coalesce(scope_resource_pat, ''),
  condition_expr,
  decision,
  priority,
  enabled,
  coalesce(created_by, ''),
  created_at,
  updated_at
from policy_rules
where ($1::bool = false or enabled = true)
order by priority asc, created_at desc
limit $2 offset $3
`, enabledOnly, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]PolicyRule, 0)
	for rows.Next() {
		var item PolicyRule
		var cond []byte
		if err := rows.Scan(
			&item.PolicyID,
			&item.Name,
			&item.Description,
			&item.ScopeAgent,
			&item.ScopeTool,
			&item.ScopeEnvironment,
			&item.ScopeResourcePat,
			&cond,
			&item.Decision,
			&item.Priority,
			&item.Enabled,
			&item.CreatedBy,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(cond, &item.ConditionExpr); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func getPolicyRuleByIDWithPool(ctx context.Context, pool policyQueryPool, policyID string) (PolicyRule, error) {
	var item PolicyRule
	var cond []byte
	err := pool.QueryRow(ctx, `
select
  policy_id,
  name,
  coalesce(description, ''),
  coalesce(scope_agent, ''),
  coalesce(scope_tool, ''),
  coalesce(scope_environment, ''),
  coalesce(scope_resource_pat, ''),
  condition_expr,
  decision,
  priority,
  enabled,
  coalesce(created_by, ''),
  created_at,
  updated_at
from policy_rules
where policy_id = $1
`, policyID).Scan(
		&item.PolicyID,
		&item.Name,
		&item.Description,
		&item.ScopeAgent,
		&item.ScopeTool,
		&item.ScopeEnvironment,
		&item.ScopeResourcePat,
		&cond,
		&item.Decision,
		&item.Priority,
		&item.Enabled,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PolicyRule{}, ErrNotFound
		}
		return PolicyRule{}, err
	}
	if err := json.Unmarshal(cond, &item.ConditionExpr); err != nil {
		return PolicyRule{}, err
	}
	return item, nil
}

func setPolicyEnabledWithPool(ctx context.Context, pool policyQueryPool, policyID string, enabled bool) error {
	tag, err := pool.Exec(ctx, `
update policy_rules
set
  enabled = $2,
  updated_at = now()
where policy_id = $1
`, policyID, enabled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type PolicyEvalResult struct {
	Decision         string
	MatchedPolicyIDs []string
	WinningRule      *PolicyRule
}

func EvaluatePolicies(rules []PolicyRule, tool, action, resource, environment, agentID, inputSummary string) PolicyEvalResult {
	if len(rules) == 0 {
		return PolicyEvalResult{Decision: "ALLOW", MatchedPolicyIDs: []string{}}
	}

	tool = strings.ToLower(strings.TrimSpace(tool))
	action = strings.ToLower(strings.TrimSpace(action))
	resource = strings.ToLower(strings.TrimSpace(resource))
	environment = strings.ToLower(strings.TrimSpace(environment))
	agentID = strings.ToLower(strings.TrimSpace(agentID))
	inputSummary = strings.ToLower(strings.TrimSpace(inputSummary))

	decisionRank := map[string]int{"ALLOW": 1, "REQUIRE_APPROVAL": 2, "BLOCK": 3}
	bestDecision := "ALLOW"
	bestRank := 1
	matched := make([]string, 0)
	var winningRule *PolicyRule

	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		if r.ScopeAgent != "" && strings.ToLower(r.ScopeAgent) != agentID {
			continue
		}
		if r.ScopeTool != "" && strings.ToLower(r.ScopeTool) != tool {
			continue
		}
		if r.ScopeEnvironment != "" && strings.ToLower(r.ScopeEnvironment) != environment {
			continue
		}
		if r.ScopeResourcePat != "" && !strings.Contains(resource, strings.ToLower(r.ScopeResourcePat)) {
			continue
		}

		if condAction, ok := r.ConditionExpr["action"].(string); ok {
			if strings.ToLower(condAction) != action {
				continue
			}
		}
		if condResourceContains, ok := r.ConditionExpr["resource_contains"].(string); ok {
			if !strings.Contains(resource, strings.ToLower(condResourceContains)) {
				continue
			}
		}
		if condPatterns, ok := r.ConditionExpr["command_patterns"].([]interface{}); ok {
			patternMatched := false
			for _, p := range condPatterns {
				if ps, ok := p.(string); ok && strings.Contains(inputSummary, strings.ToLower(ps)) {
					patternMatched = true
					break
				}
			}
			if !patternMatched {
				continue
			}
		}

		matched = append(matched, r.PolicyID)
		if rank := decisionRank[r.Decision]; rank > bestRank {
			bestRank = rank
			bestDecision = r.Decision
			rCopy := r
			winningRule = &rCopy
		}
	}

	if len(matched) == 0 {
		return PolicyEvalResult{Decision: "ALLOW", MatchedPolicyIDs: []string{}}
	}
	return PolicyEvalResult{
		Decision:         bestDecision,
		MatchedPolicyIDs: matched,
		WinningRule:      winningRule,
	}
}

func EvaluatePolicyPreview(rules []PolicyRule, tool, action, resource, environment string) (string, []string) {
	result := EvaluatePolicies(rules, tool, action, resource, environment, "", "")
	return result.Decision, result.MatchedPolicyIDs
}
