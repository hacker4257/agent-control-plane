package repo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type fakePolicyRows struct {
	items []PolicyRule
	idx   int
}

func (r *fakePolicyRows) Next() bool {
	if r.idx >= len(r.items) {
		return false
	}
	r.idx++
	return true
}

func (r *fakePolicyRows) Scan(dest ...interface{}) error {
	item := r.items[r.idx-1]
	cond, _ := json.Marshal(item.ConditionExpr)
	*(dest[0].(*string)) = item.PolicyID
	*(dest[1].(*string)) = item.Name
	*(dest[2].(*string)) = item.Description
	*(dest[3].(*string)) = item.ScopeAgent
	*(dest[4].(*string)) = item.ScopeTool
	*(dest[5].(*string)) = item.ScopeEnvironment
	*(dest[6].(*string)) = item.ScopeResourcePat
	*(dest[7].(*[]byte)) = cond
	*(dest[8].(*string)) = item.Decision
	*(dest[9].(*int)) = item.Priority
	*(dest[10].(*bool)) = item.Enabled
	*(dest[11].(*string)) = item.CreatedBy
	*(dest[12].(*time.Time)) = item.CreatedAt
	*(dest[13].(*time.Time)) = item.UpdatedAt
	return nil
}

func (r *fakePolicyRows) Err() error { return nil }
func (r *fakePolicyRows) Close()     {}

type fakePolicyExecTag struct{ n int64 }

func (t fakePolicyExecTag) RowsAffected() int64 { return t.n }

type fakePolicyPool struct {
	lastQuerySQL string
	lastExecSQL  string
	lastExecArgs []interface{}
	rows         *fakePolicyRows
	queryErr     error
	execErr      error
	execRows     int64
	queryRowData PolicyRule
	queryRowErr  error
}

func (p *fakePolicyPool) Query(ctx context.Context, sql string, args ...interface{}) (policyRowsIface, error) {
	p.lastQuerySQL = sql
	if p.queryErr != nil {
		return nil, p.queryErr
	}
	if p.rows == nil {
		p.rows = &fakePolicyRows{items: []PolicyRule{}}
	}
	return p.rows, nil
}

func (p *fakePolicyPool) QueryRow(ctx context.Context, sql string, args ...interface{}) policyRowIface {
	cond, _ := json.Marshal(p.queryRowData.ConditionExpr)
	createdAt := p.queryRowData.CreatedAt
	updatedAt := p.queryRowData.UpdatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	vals := []interface{}{
		p.queryRowData.PolicyID,
		p.queryRowData.Name,
		p.queryRowData.Description,
		p.queryRowData.ScopeAgent,
		p.queryRowData.ScopeTool,
		p.queryRowData.ScopeEnvironment,
		p.queryRowData.ScopeResourcePat,
		[]byte(cond),
		p.queryRowData.Decision,
		p.queryRowData.Priority,
		p.queryRowData.Enabled,
		p.queryRowData.CreatedBy,
		createdAt,
		updatedAt,
	}
	return fakePolicyRow{vals: vals, err: p.queryRowErr}
}

func (p *fakePolicyPool) Exec(ctx context.Context, sql string, args ...interface{}) (policyExecTagIface, error) {
	p.lastExecSQL = sql
	p.lastExecArgs = args
	if p.execErr != nil {
		return nil, p.execErr
	}
	return fakePolicyExecTag{n: p.execRows}, nil
}

type fakePolicyRow struct {
	vals []interface{}
	err  error
}

func (r fakePolicyRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.vals[i].(string)
		case *int:
			*d = r.vals[i].(int)
		case *bool:
			*d = r.vals[i].(bool)
		case *[]byte:
			*d = r.vals[i].([]byte)
		case *time.Time:
			*d = r.vals[i].(time.Time)
		default:
			return errors.New("unsupported scan type")
		}
	}
	return nil
}

func TestListPolicyRulesQueryIncludesOrderByPriority(t *testing.T) {
	pool := &fakePolicyPool{
		rows: &fakePolicyRows{items: []PolicyRule{{
			PolicyID:      "pol_1",
			Name:          "Protect main",
			Decision:      "REQUIRE_APPROVAL",
			ConditionExpr: map[string]interface{}{"action": "push"},
			Priority:      10,
			Enabled:       true,
			CreatedAt:     time.Now().UTC(),
			UpdatedAt:     time.Now().UTC(),
		}}},
	}
	items, err := listPolicyRulesWithPool(context.Background(), pool, true, 20, 0)
	if err != nil {
		t.Fatalf("list policy rules: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(items))
	}
	if !strings.Contains(strings.ToLower(pool.lastQuerySQL), "order by priority asc") {
		t.Fatalf("expected order by priority asc in query, got: %s", pool.lastQuerySQL)
	}
}

func TestGetPolicyRuleByIDMapsNoRows(t *testing.T) {
	pool := &fakePolicyPool{queryRowErr: pgx.ErrNoRows}
	_, err := getPolicyRuleByIDWithPool(context.Background(), pool, "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSetPolicyEnabledUpdatesByID(t *testing.T) {
	pool := &fakePolicyPool{execRows: 1}
	err := setPolicyEnabledWithPool(context.Background(), pool, "pol_1", true)
	if err != nil {
		t.Fatalf("set policy enabled: %v", err)
	}
	if !strings.Contains(strings.ToLower(pool.lastExecSQL), "update policy_rules") {
		t.Fatalf("expected update statement, got: %s", pool.lastExecSQL)
	}
	if len(pool.lastExecArgs) == 0 || pool.lastExecArgs[0] != "pol_1" {
		t.Fatalf("expected first arg policy id pol_1")
	}
}

func TestSetPolicyEnabledMapsNoRowsToNotFound(t *testing.T) {
	pool := &fakePolicyPool{execRows: 0}
	err := setPolicyEnabledWithPool(context.Background(), pool, "missing", false)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPolicyEvaluatePreviewReturnsAllowWhenNoMatch(t *testing.T) {
	decision, matched := EvaluatePolicyPreview([]PolicyRule{}, "shell", "exec", "host:prod", "dev")
	if decision != "ALLOW" {
		t.Fatalf("expected ALLOW, got %s", decision)
	}
	if len(matched) != 0 {
		t.Fatalf("expected no matched policy ids")
	}
}

func TestPolicyEvaluatePreviewMatchesRequireApproval(t *testing.T) {
	rules := []PolicyRule{{
		PolicyID:         "pol_1",
		Decision:         "REQUIRE_APPROVAL",
		ScopeTool:        "github",
		ScopeEnvironment: "prod",
		ScopeResourcePat: "branch:main",
		ConditionExpr:    map[string]interface{}{"action": "push"},
		Priority:         10,
		Enabled:          true,
	}}
	decision, matched := EvaluatePolicyPreview(rules, "github", "push", "repo:x/branch:main", "prod")
	if decision != "REQUIRE_APPROVAL" {
		t.Fatalf("expected REQUIRE_APPROVAL, got %s", decision)
	}
	if len(matched) != 1 || matched[0] != "pol_1" {
		t.Fatalf("expected matched pol_1, got %+v", matched)
	}
}

func TestEvaluatePoliciesCommandPatterns(t *testing.T) {
	rules := []PolicyRule{
		{
			PolicyID:      "pol_shell_block",
			ScopeTool:     "shell",
			ConditionExpr: map[string]interface{}{"command_patterns": []interface{}{"rm -rf", "curl|sh"}},
			Decision:      "BLOCK",
			Priority:      5,
			Enabled:       true,
		},
	}
	result := EvaluatePolicies(rules, "shell", "exec", "host:prod", "prod", "my-agent", "rm -rf /tmp/cache")
	if result.Decision != "BLOCK" {
		t.Fatalf("expected BLOCK, got %s", result.Decision)
	}
	if len(result.MatchedPolicyIDs) != 1 || result.MatchedPolicyIDs[0] != "pol_shell_block" {
		t.Fatalf("expected matched pol_shell_block, got %+v", result.MatchedPolicyIDs)
	}
	if result.WinningRule == nil || result.WinningRule.PolicyID != "pol_shell_block" {
		t.Fatal("expected WinningRule to be set")
	}
}

func TestEvaluatePoliciesNoCommandPatternMatch(t *testing.T) {
	rules := []PolicyRule{
		{
			PolicyID:      "pol_shell_block",
			ScopeTool:     "shell",
			ConditionExpr: map[string]interface{}{"command_patterns": []interface{}{"rm -rf", "curl|sh"}},
			Decision:      "BLOCK",
			Priority:      5,
			Enabled:       true,
		},
	}
	result := EvaluatePolicies(rules, "shell", "exec", "host:prod", "prod", "my-agent", "ls -la /tmp")
	if result.Decision != "ALLOW" {
		t.Fatalf("expected ALLOW, got %s", result.Decision)
	}
}

func TestEvaluatePoliciesScopeAgent(t *testing.T) {
	rules := []PolicyRule{
		{
			PolicyID:   "pol_agent_block",
			ScopeAgent: "risky-agent",
			Decision:   "BLOCK",
			Priority:   1,
			Enabled:    true,
		},
	}
	result := EvaluatePolicies(rules, "shell", "exec", "host:prod", "prod", "risky-agent", "ls")
	if result.Decision != "BLOCK" {
		t.Fatalf("expected BLOCK, got %s", result.Decision)
	}

	result = EvaluatePolicies(rules, "shell", "exec", "host:prod", "prod", "safe-agent", "ls")
	if result.Decision != "ALLOW" {
		t.Fatalf("expected ALLOW for non-matching agent, got %s", result.Decision)
	}
}
