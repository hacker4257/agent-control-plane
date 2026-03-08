package service

import (
	"context"
	"strings"

	"agent-control-plane/apps/api/internal/repo"
	"github.com/google/uuid"
)

type GatewayStore interface {
	InsertEvent(ctx context.Context, e repo.EventRecord) error
	UpsertSessionProjection(ctx context.Context, u repo.SessionProjectionUpdate) error
	CreateApproval(ctx context.Context, a repo.ApprovalRecord) error
	ListPolicyRules(ctx context.Context, enabledOnly bool, limit, offset int) ([]repo.PolicyRule, error)
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
	if s == nil || s.store == nil {
		return defaultAllowResponse(), nil
	}

	resp, err := s.evaluatePreflight(ctx, in)
	if err != nil {
		return PreflightOutput{}, err
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

func (s *GatewayService) evaluatePreflight(ctx context.Context, in PreflightInput) (PreflightOutput, error) {
	rules, err := s.store.ListPolicyRules(ctx, true, 200, 0)
	if err != nil {
		return PreflightOutput{}, err
	}

	result := repo.EvaluatePolicies(
		rules,
		in.Tool,
		in.Action,
		in.Resource,
		in.Environment,
		in.AgentID,
		in.InputSummary+" "+in.Command,
	)

	out := PreflightOutput{
		Decision:         result.Decision,
		DecisionID:       "dec_" + lower(result.Decision),
		MatchedPolicyIDs: result.MatchedPolicyIDs,
		ReasonCode:       "DEFAULT_ALLOW",
		ReasonText:       "No policy matched",
		RiskTags:         []string{},
	}

	if result.WinningRule != nil {
		out.ReasonCode = result.WinningRule.PolicyID
		out.ReasonText = result.WinningRule.Name
		if result.WinningRule.Description != "" {
			out.ReasonText = result.WinningRule.Description
		}
		tags := []string{}
		if t := lower(in.Tool); t != "" {
			tags = append(tags, t)
		}
		if result.Decision == "BLOCK" {
			tags = append(tags, "blocked_action")
		} else if result.Decision == "REQUIRE_APPROVAL" {
			tags = append(tags, "approval_required")
		}
		out.RiskTags = tags
	}

	return out, nil
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

func defaultAllowResponse() PreflightOutput {
	return PreflightOutput{
		Decision:         "ALLOW",
		DecisionID:       "dec_allow",
		MatchedPolicyIDs: []string{},
		ReasonCode:       "DEFAULT_ALLOW",
		ReasonText:       "No policy matched",
		RiskTags:         []string{},
	}
}

func lower(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
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
