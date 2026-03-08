package service

import (
	"context"
	"strings"

	"agent-control-plane/apps/api/internal/repo"
	"github.com/google/uuid"
)

const (
	policyShellDangerous = "pol_deny_shell_dangerous_commands"
	policyGithubMain     = "pol_github_main_branch_protection"
	policyBrowserPay     = "pol_browser_financial_action_guard"
)

type GatewayStore interface {
	InsertEvent(ctx context.Context, e repo.EventRecord) error
	UpsertSessionProjection(ctx context.Context, u repo.SessionProjectionUpdate) error
	CreateApproval(ctx context.Context, a repo.ApprovalRecord) error
}

type GatewayService struct {
	store GatewayStore
}

type PreflightInput struct {
	SessionID     string
	StepID        string
	CorrelationID string
	AgentID       string
	UserID        string
	Environment   string
	Objective     string
	Tool          string
	Action        string
	Resource      string
	InputSummary  string
	Command       string
}

type PreflightOutput struct {
	Decision         string   `json:"decision"`
	DecisionID       string   `json:"decision_id"`
	MatchedPolicyIDs []string `json:"matched_policy_ids"`
	ReasonCode       string   `json:"reason_code"`
	ReasonText       string   `json:"reason_text"`
	RiskTags         []string `json:"risk_tags"`
	ApprovalID       string   `json:"approval_id,omitempty"`
}

type PostflightInput struct {
	SessionID     string
	StepID        string
	CorrelationID string
	AgentID       string
	Environment   string
	Tool          string
	Action        string
	Resource      string
	Result        string
	OutputSummary string
	ArtifactRefs  []string
}

func NewGatewayService(store GatewayStore) *GatewayService {
	return &GatewayService{store: store}
}

func (s *GatewayService) ProcessPreflight(ctx context.Context, in PreflightInput) (PreflightOutput, error) {
	resp := decidePreflight(in)
	if s == nil || s.store == nil {
		return resp, nil
	}

	eventID := uuid.NewString()
	e := repo.EventRecord{
		EventID:          eventID,
		SessionID:        in.SessionID,
		StepID:           in.StepID,
		CorrelationID:    in.CorrelationID,
		EventType:        eventTypeForDecision(resp.Decision),
		Decision:         resp.Decision,
		Tool:             lower(in.Tool),
		Action:           lower(in.Action),
		Resource:         in.Resource,
		RiskScore:        riskScoreFromDecision(resp.Decision),
		RiskTags:         resp.RiskTags,
		MatchedPolicyIDs: resp.MatchedPolicyIDs,
		ReasonCode:       resp.ReasonCode,
		ReasonText:       resp.ReasonText,
		InputSummary:     in.InputSummary,
		ActorType:        "agent",
		ActorID:          in.AgentID,
	}
	if err := s.store.InsertEvent(ctx, e); err != nil {
		return PreflightOutput{}, err
	}

	if err := s.store.UpsertSessionProjection(ctx, repo.SessionProjectionUpdate{
		SessionID:   in.SessionID,
		AgentID:     in.AgentID,
		UserID:      in.UserID,
		Environment: in.Environment,
		Objective:   in.Objective,
		Status:      sessionStatusFromDecision(resp.Decision),
		RiskScore:   riskScoreFromDecision(resp.Decision),
		Resource:    in.Resource,
		IsApproval:  resp.Decision == "REQUIRE_APPROVAL",
		IsBlocked:   resp.Decision == "BLOCK",
	}); err != nil {
		return PreflightOutput{}, err
	}

	if resp.Decision == "REQUIRE_APPROVAL" {
		approvalID := resp.ApprovalID
		if approvalID == "" {
			approvalID = "appr_" + uuid.NewString()
			resp.ApprovalID = approvalID
		}
		if err := s.store.CreateApproval(ctx, repo.ApprovalRecord{
			ApprovalID:      approvalID,
			SessionID:       in.SessionID,
			StepID:          in.StepID,
			EventID:         eventID,
			Status:          "pending",
			Action:          lower(in.Action),
			Tool:            lower(in.Tool),
			Resource:        in.Resource,
			Objective:       in.Objective,
			TriggerReason:   resp.ReasonText,
			RiskTags:        resp.RiskTags,
			PotentialImpact: "Protected operation in sensitive environment",
		}); err != nil {
			return PreflightOutput{}, err
		}
	}

	return resp, nil
}

func (s *GatewayService) ProcessPostflight(ctx context.Context, in PostflightInput) error {
	if s == nil || s.store == nil {
		return nil
	}

	e := repo.EventRecord{
		EventID:       uuid.NewString(),
		SessionID:     in.SessionID,
		StepID:        in.StepID,
		CorrelationID: in.CorrelationID,
		EventType:     eventTypeForPostflightResult(in.Result),
		Tool:          lower(in.Tool),
		Action:        lower(in.Action),
		Resource:      in.Resource,
		OutputSummary: in.OutputSummary,
		ArtifactRefs:  in.ArtifactRefs,
		ActorType:     "agent",
		ActorID:       in.AgentID,
	}
	if err := s.store.InsertEvent(ctx, e); err != nil {
		return err
	}

	return s.store.UpsertSessionProjection(ctx, repo.SessionProjectionUpdate{
		SessionID:   in.SessionID,
		AgentID:     in.AgentID,
		Environment: in.Environment,
		Objective:   "Agent task",
		Status:      sessionStatusFromPostflightResult(in.Result),
		RiskScore:   riskScoreFromPostflightResult(in.Result),
		Resource:    in.Resource,
		IsApproval:  false,
		IsBlocked:   lower(in.Result) == "failed",
	})
}

func decidePreflight(in PreflightInput) PreflightOutput {
	tool := lower(in.Tool)
	action := lower(in.Action)
	resource := lower(in.Resource)
	inputSummary := lower(in.InputSummary)
	command := lower(in.Command)
	environment := lower(in.Environment)

	if tool == "shell" && isDangerousShell(inputSummary+" "+command) {
		return PreflightOutput{
			Decision:         "BLOCK",
			DecisionID:       "dec_block_shell_dangerous",
			MatchedPolicyIDs: []string{policyShellDangerous},
			ReasonCode:       "SHELL_DANGEROUS_COMMAND",
			ReasonText:       "Dangerous shell command pattern detected",
			RiskTags:         []string{"destructive_action", "shell"},
		}
	}

	if tool == "github" && action == "push" && environment == "prod" && strings.Contains(resource, "branch:main") {
		return PreflightOutput{
			Decision:         "REQUIRE_APPROVAL",
			DecisionID:       "dec_require_approval_github_main",
			MatchedPolicyIDs: []string{policyGithubMain},
			ReasonCode:       "PROTECTED_BRANCH_WRITE",
			ReasonText:       "Push to main branch in prod requires approval",
			RiskTags:         []string{"repo_write", "protected_branch"},
			ApprovalID:       "appr_github_main_001",
		}
	}

	if tool == "browser" && action == "submit" && strings.Contains(resource, "payment") {
		return PreflightOutput{
			Decision:         "BLOCK",
			DecisionID:       "dec_block_browser_payment",
			MatchedPolicyIDs: []string{policyBrowserPay},
			ReasonCode:       "FINANCIAL_SUBMIT_BLOCKED",
			ReasonText:       "Browser submit on payment resource is blocked",
			RiskTags:         []string{"financial_action", "browser"},
		}
	}

	return PreflightOutput{
		Decision:         "ALLOW",
		DecisionID:       "dec_default_allow",
		MatchedPolicyIDs: []string{},
		ReasonCode:       "DEFAULT_ALLOW",
		ReasonText:       "No policy matched",
		RiskTags:         []string{},
	}
}

func lower(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func isDangerousShell(v string) bool {
	patterns := []string{"rm -rf", "sudo", "curl | sh", "curl|sh"}
	for _, p := range patterns {
		if strings.Contains(v, p) {
			return true
		}
	}
	return false
}

func eventTypeForDecision(decision string) string {
	switch decision {
	case "BLOCK":
		return "policy_blocked"
	case "REQUIRE_APPROVAL":
		return "approval_requested"
	default:
		return "tool_requested"
	}
}

func eventTypeForPostflightResult(result string) string {
	switch lower(result) {
	case "failed":
		return "tool_failed"
	default:
		return "tool_completed"
	}
}

func sessionStatusFromDecision(decision string) string {
	switch decision {
	case "BLOCK":
		return "blocked"
	case "REQUIRE_APPROVAL":
		return "approval_pending"
	default:
		return "running"
	}
}

func sessionStatusFromPostflightResult(result string) string {
	switch lower(result) {
	case "failed":
		return "blocked"
	default:
		return "completed"
	}
}

func riskScoreFromDecision(decision string) int {
	switch decision {
	case "BLOCK":
		return 95
	case "REQUIRE_APPROVAL":
		return 85
	default:
		return 30
	}
}

func riskScoreFromPostflightResult(result string) int {
	switch lower(result) {
	case "failed":
		return 60
	default:
		return 10
	}
}
