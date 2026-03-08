package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

type EventRecord struct {
	EventID          string
	SessionID        string
	StepID           string
	CorrelationID    string
	EventType        string
	Decision         string
	Tool             string
	Action           string
	Resource         string
	RiskScore        int
	RiskTags         []string
	MatchedPolicyIDs []string
	ReasonCode       string
	ReasonText       string
	InputSummary     string
	OutputSummary    string
	ArtifactRefs     []string
	ActorType        string
	ActorID          string
	CreatedAt        time.Time
}

type SessionRecord struct {
	SessionID        string
	Objective        string
	AgentID          string
	UserID           string
	Environment      string
	Status           string
	StartedAt        time.Time
	EndedAt          *time.Time
	RiskScore        int
	ApprovalsCount   int
	BlockedCount     int
	TouchedResources []string
	LastEventAt      *time.Time
	UpdatedAt        time.Time
}

type SessionProjectionUpdate struct {
	SessionID   string
	Objective   string
	AgentID     string
	UserID      string
	Environment string
	Status      string
	EventTime   time.Time
	RiskScore   int
	IsApproval  bool
	IsBlocked   bool
	Resource    string
}

type DashboardSummary struct {
	SessionsCount         int `json:"sessions_count"`
	PendingApprovalsCount int `json:"pending_approvals_count"`
	BlockedActionsCount   int `json:"blocked_actions_count"`
	PolicyHitsCount       int `json:"policy_hits_count"`
}

type ApprovalRecord struct {
	ApprovalID        string
	SessionID         string
	StepID            string
	EventID           string
	Status            string
	Action            string
	Tool              string
	Resource          string
	Objective         string
	TriggerReason     string
	RiskTags          []string
	PotentialImpact   string
	SuggestedSafeAlt  string
	RequestedAt       time.Time
	DecidedAt         *time.Time
	ApproverID        string
	DecisionComment   string
}

type ApprovalDecisionInput struct {
	ApprovalID      string
	Decision        string
	ApproverID      string
	DecisionComment string
}

var ErrNotFound = errors.New("not found")

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) InsertEvent(ctx context.Context, e EventRecord) error {
	riskTags, err := json.Marshal(e.RiskTags)
	if err != nil {
		return err
	}
	policyIDs, err := json.Marshal(e.MatchedPolicyIDs)
	if err != nil {
		return err
	}
	artifactRefs, err := json.Marshal(e.ArtifactRefs)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx, `
insert into tool_events (
  event_id,
  session_id,
  step_id,
  correlation_id,
  event_type,
  decision,
  tool,
  action,
  resource,
  risk_score,
  risk_tags,
  matched_policy_ids,
  reason_code,
  reason_text,
  input_summary,
  output_summary,
  artifact_refs,
  actor_type,
  actor_id
) values (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12::jsonb,$13,$14,$15,$16,$17::jsonb,$18,$19
)
`,
		e.EventID,
		e.SessionID,
		nullIfEmpty(e.StepID),
		nullIfEmpty(e.CorrelationID),
		e.EventType,
		nullIfEmpty(e.Decision),
		nullIfEmpty(e.Tool),
		nullIfEmpty(e.Action),
		nullIfEmpty(e.Resource),
		e.RiskScore,
		string(riskTags),
		string(policyIDs),
		nullIfEmpty(e.ReasonCode),
		nullIfEmpty(e.ReasonText),
		nullIfEmpty(e.InputSummary),
		nullIfEmpty(e.OutputSummary),
		string(artifactRefs),
		nullIfEmpty(e.ActorType),
		nullIfEmpty(e.ActorID),
	)
	return err
}

func (s *Store) UpsertSessionProjection(ctx context.Context, u SessionProjectionUpdate) error {
	_, err := s.pool.Exec(ctx, `
insert into sessions (
  session_id,
  objective,
  agent_id,
  user_id,
  environment,
  status,
  started_at,
  ended_at,
  risk_score,
  approvals_count,
  blocked_count,
  touched_resources,
  last_event_at,
  updated_at
) values (
  $1,
  $2,
  $3,
  $4,
  $5,
  $6,
  coalesce($7, now()),
  case when $6 in ('completed', 'blocked') then coalesce($7, now()) else null end,
  $8,
  case when $9 then 1 else 0 end,
  case when $10 then 1 else 0 end,
  case when $11 = '' then '[]'::jsonb else to_jsonb(array[$11]) end,
  coalesce($7, now()),
  now()
)
on conflict (session_id) do update set
  status = excluded.status,
  risk_score = greatest(sessions.risk_score, excluded.risk_score),
  approvals_count = sessions.approvals_count + case when $9 then 1 else 0 end,
  blocked_count = sessions.blocked_count + case when $10 then 1 else 0 end,
  touched_resources = case
    when $11 = '' then sessions.touched_resources
    else coalesce(sessions.touched_resources, '[]'::jsonb) || to_jsonb(array[$11])
  end,
  last_event_at = coalesce($7, now()),
  ended_at = case
    when excluded.status in ('completed', 'blocked') then coalesce($7, now())
    else sessions.ended_at
  end,
  updated_at = now();
`,
		u.SessionID,
		defaultIfEmpty(u.Objective, "Agent task"),
		defaultIfEmpty(u.AgentID, "unknown-agent"),
		nullIfEmpty(u.UserID),
		defaultIfEmpty(u.Environment, "unknown"),
		defaultIfEmpty(u.Status, "running"),
		nullTimeIfZero(u.EventTime),
		u.RiskScore,
		u.IsApproval,
		u.IsBlocked,
		u.Resource,
	)
	return err
}

func (s *Store) ListSessions(ctx context.Context, limit, offset int) ([]SessionRecord, error) {
	rows, err := s.pool.Query(ctx, `
select
  session_id,
  objective,
  agent_id,
  coalesce(user_id, ''),
  environment,
  status,
  started_at,
  ended_at,
  risk_score,
  approvals_count,
  blocked_count,
  coalesce(touched_resources, '[]'::jsonb),
  last_event_at,
  updated_at
from sessions
order by coalesce(last_event_at, started_at) desc
limit $1 offset $2
`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]SessionRecord, 0)
	for rows.Next() {
		var srec SessionRecord
		var userID string
		var touched []byte
		if err := rows.Scan(
			&srec.SessionID,
			&srec.Objective,
			&srec.AgentID,
			&userID,
			&srec.Environment,
			&srec.Status,
			&srec.StartedAt,
			&srec.EndedAt,
			&srec.RiskScore,
			&srec.ApprovalsCount,
			&srec.BlockedCount,
			&touched,
			&srec.LastEventAt,
			&srec.UpdatedAt,
		); err != nil {
			return nil, err
		}
		srec.UserID = userID
		if err := json.Unmarshal(touched, &srec.TouchedResources); err != nil {
			return nil, err
		}
		items = append(items, srec)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) GetSessionByID(ctx context.Context, sessionID string) (SessionRecord, error) {
	var srec SessionRecord
	var userID string
	var touched []byte
	err := s.pool.QueryRow(ctx, `
select
  session_id,
  objective,
  agent_id,
  coalesce(user_id, ''),
  environment,
  status,
  started_at,
  ended_at,
  risk_score,
  approvals_count,
  blocked_count,
  coalesce(touched_resources, '[]'::jsonb),
  last_event_at,
  updated_at
from sessions
where session_id = $1
`, sessionID).Scan(
		&srec.SessionID,
		&srec.Objective,
		&srec.AgentID,
		&userID,
		&srec.Environment,
		&srec.Status,
		&srec.StartedAt,
		&srec.EndedAt,
		&srec.RiskScore,
		&srec.ApprovalsCount,
		&srec.BlockedCount,
		&touched,
		&srec.LastEventAt,
		&srec.UpdatedAt,
	)
	if err != nil {
		return SessionRecord{}, err
	}
	srec.UserID = userID
	if err := json.Unmarshal(touched, &srec.TouchedResources); err != nil {
		return SessionRecord{}, err
	}
	return srec, nil
}

func (s *Store) GetDashboardSummary(ctx context.Context) (DashboardSummary, error) {
	var summary DashboardSummary
	err := s.pool.QueryRow(ctx, `
select
  (select count(*)::int from sessions) as sessions_count,
  (select count(*)::int from approvals where status = 'pending') as pending_approvals_count,
  (select count(*)::int from tool_events where event_type = 'blocked') as blocked_actions_count,
  (select count(*)::int from tool_events where coalesce(jsonb_array_length(matched_policy_ids), 0) > 0) as policy_hits_count
`).Scan(
		&summary.SessionsCount,
		&summary.PendingApprovalsCount,
		&summary.BlockedActionsCount,
		&summary.PolicyHitsCount,
	)
	if err != nil {
		return DashboardSummary{}, err
	}
	return summary, nil
}

func (s *Store) CreateApproval(ctx context.Context, a ApprovalRecord) error {
	riskTags, err := json.Marshal(a.RiskTags)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
insert into approvals (
  approval_id,
  session_id,
  step_id,
  event_id,
  status,
  action,
  tool,
  resource,
  objective,
  trigger_reason,
  risk_tags,
  potential_impact,
  suggested_safe_alt,
  requested_at,
  decided_at,
  approver_id,
  decision_comment
) values (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$13,
  coalesce($14, now()),$15,$16,$17
)
on conflict (approval_id) do nothing
`,
		a.ApprovalID,
		a.SessionID,
		nullIfEmpty(a.StepID),
		nullIfEmpty(a.EventID),
		defaultIfEmpty(a.Status, "pending"),
		defaultIfEmpty(a.Action, "unknown"),
		defaultIfEmpty(a.Tool, "unknown"),
		defaultIfEmpty(a.Resource, "unknown"),
		nullIfEmpty(a.Objective),
		defaultIfEmpty(a.TriggerReason, "approval required"),
		string(riskTags),
		nullIfEmpty(a.PotentialImpact),
		nullIfEmpty(a.SuggestedSafeAlt),
		nullTimeIfZero(a.RequestedAt),
		a.DecidedAt,
		nullIfEmpty(a.ApproverID),
		nullIfEmpty(a.DecisionComment),
	)
	return err
}

func (s *Store) ListApprovals(ctx context.Context, status string, limit, offset int) ([]ApprovalRecord, error) {
	rows, err := s.pool.Query(ctx, `
select
  approval_id,
  session_id,
  coalesce(step_id, ''),
  coalesce(event_id, ''),
  status,
  action,
  tool,
  resource,
  coalesce(objective, ''),
  trigger_reason,
  coalesce(risk_tags, '[]'::jsonb),
  coalesce(potential_impact, ''),
  coalesce(suggested_safe_alt, ''),
  requested_at,
  decided_at,
  coalesce(approver_id, ''),
  coalesce(decision_comment, '')
from approvals
where ($1 = '' or status = $1)
order by requested_at desc
limit $2 offset $3
`, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ApprovalRecord, 0)
	for rows.Next() {
		var a ApprovalRecord
		var riskTags []byte
		if err := rows.Scan(
			&a.ApprovalID,
			&a.SessionID,
			&a.StepID,
			&a.EventID,
			&a.Status,
			&a.Action,
			&a.Tool,
			&a.Resource,
			&a.Objective,
			&a.TriggerReason,
			&riskTags,
			&a.PotentialImpact,
			&a.SuggestedSafeAlt,
			&a.RequestedAt,
			&a.DecidedAt,
			&a.ApproverID,
			&a.DecisionComment,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(riskTags, &a.RiskTags); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) GetApprovalByID(ctx context.Context, approvalID string) (ApprovalRecord, error) {
	var a ApprovalRecord
	var riskTags []byte
	err := s.pool.QueryRow(ctx, `
select
  approval_id,
  session_id,
  coalesce(step_id, ''),
  coalesce(event_id, ''),
  status,
  action,
  tool,
  resource,
  coalesce(objective, ''),
  trigger_reason,
  coalesce(risk_tags, '[]'::jsonb),
  coalesce(potential_impact, ''),
  coalesce(suggested_safe_alt, ''),
  requested_at,
  decided_at,
  coalesce(approver_id, ''),
  coalesce(decision_comment, '')
from approvals
where approval_id = $1
`, approvalID).Scan(
		&a.ApprovalID,
		&a.SessionID,
		&a.StepID,
		&a.EventID,
		&a.Status,
		&a.Action,
		&a.Tool,
		&a.Resource,
		&a.Objective,
		&a.TriggerReason,
		&riskTags,
		&a.PotentialImpact,
		&a.SuggestedSafeAlt,
		&a.RequestedAt,
		&a.DecidedAt,
		&a.ApproverID,
		&a.DecisionComment,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ApprovalRecord{}, ErrNotFound
		}
		return ApprovalRecord{}, err
	}
	if err := json.Unmarshal(riskTags, &a.RiskTags); err != nil {
		return ApprovalRecord{}, err
	}
	return a, nil
}

func (s *Store) ApplyApprovalDecision(ctx context.Context, in ApprovalDecisionInput) (ApprovalRecord, error) {
	now := time.Now().UTC()
	status := "approved"
	sessionStatus := "running"
	isBlocked := false
	if in.Decision == "reject" {
		status = "rejected"
		sessionStatus = "blocked"
		isBlocked = true
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ApprovalRecord{}, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
update approvals
set
  status = $2,
  decided_at = $3,
  approver_id = $4,
  decision_comment = $5
where approval_id = $1 and status = 'pending'
`, in.ApprovalID, status, now, nullIfEmpty(in.ApproverID), nullIfEmpty(in.DecisionComment))
	if err != nil {
		return ApprovalRecord{}, err
	}
	if tag.RowsAffected() == 0 {
		return ApprovalRecord{}, ErrNotFound
	}

	var approval ApprovalRecord
	var riskTags []byte
	err = tx.QueryRow(ctx, `
select
  approval_id,
  session_id,
  coalesce(step_id, ''),
  coalesce(event_id, ''),
  status,
  action,
  tool,
  resource,
  coalesce(objective, ''),
  trigger_reason,
  coalesce(risk_tags, '[]'::jsonb),
  coalesce(potential_impact, ''),
  coalesce(suggested_safe_alt, ''),
  requested_at,
  decided_at,
  coalesce(approver_id, ''),
  coalesce(decision_comment, '')
from approvals
where approval_id = $1
`, in.ApprovalID).Scan(
		&approval.ApprovalID,
		&approval.SessionID,
		&approval.StepID,
		&approval.EventID,
		&approval.Status,
		&approval.Action,
		&approval.Tool,
		&approval.Resource,
		&approval.Objective,
		&approval.TriggerReason,
		&riskTags,
		&approval.PotentialImpact,
		&approval.SuggestedSafeAlt,
		&approval.RequestedAt,
		&approval.DecidedAt,
		&approval.ApproverID,
		&approval.DecisionComment,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ApprovalRecord{}, ErrNotFound
		}
		return ApprovalRecord{}, err
	}
	if err := json.Unmarshal(riskTags, &approval.RiskTags); err != nil {
		return ApprovalRecord{}, err
	}

	_, err = tx.Exec(ctx, `
update sessions
set
  status = $2,
  blocked_count = blocked_count + case when $3 then 1 else 0 end,
  last_event_at = $4,
  ended_at = case when $2 = 'blocked' then $4 else ended_at end,
  updated_at = $4
where session_id = $1
`, approval.SessionID, sessionStatus, isBlocked, now)
	if err != nil {
		return ApprovalRecord{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return ApprovalRecord{}, err
	}

	return approval, nil
}

func (s *Store) ListSessionEvents(ctx context.Context, sessionID string, limit, offset int) ([]EventRecord, error) {
	rows, err := s.pool.Query(ctx, `
select
  event_id,
  session_id,
  coalesce(step_id, ''),
  coalesce(correlation_id, ''),
  event_type,
  coalesce(decision, ''),
  coalesce(tool, ''),
  coalesce(action, ''),
  coalesce(resource, ''),
  coalesce(risk_score, 0),
  coalesce(risk_tags, '[]'::jsonb),
  coalesce(matched_policy_ids, '[]'::jsonb),
  coalesce(reason_code, ''),
  coalesce(reason_text, ''),
  coalesce(input_summary, ''),
  coalesce(output_summary, ''),
  coalesce(artifact_refs, '[]'::jsonb),
  coalesce(actor_type, ''),
  coalesce(actor_id, ''),
  created_at
from tool_events
where session_id = $1
order by created_at asc
limit $2 offset $3
`, sessionID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]EventRecord, 0)
	for rows.Next() {
		var e EventRecord
		var riskTags []byte
		var policyIDs []byte
		var artifactRefs []byte
		if err := rows.Scan(
			&e.EventID,
			&e.SessionID,
			&e.StepID,
			&e.CorrelationID,
			&e.EventType,
			&e.Decision,
			&e.Tool,
			&e.Action,
			&e.Resource,
			&e.RiskScore,
			&riskTags,
			&policyIDs,
			&e.ReasonCode,
			&e.ReasonText,
			&e.InputSummary,
			&e.OutputSummary,
			&artifactRefs,
			&e.ActorType,
			&e.ActorID,
			&e.CreatedAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(riskTags, &e.RiskTags); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(policyIDs, &e.MatchedPolicyIDs); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(artifactRefs, &e.ArtifactRefs); err != nil {
			return nil, err
		}
		items = append(items, e)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func nullIfEmpty(v string) interface{} {
	if v == "" {
		return nil
	}
	return v
}

func defaultIfEmpty(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func nullTimeIfZero(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t
}
