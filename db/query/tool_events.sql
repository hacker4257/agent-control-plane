-- name: InsertToolEvent :one
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
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19
)
returning *;

-- name: ListSessionEvents :many
select *
from tool_events
where session_id = $1
order by created_at asc
limit $2 offset $3;
