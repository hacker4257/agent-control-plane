create table tool_events (
  event_id           text primary key,
  session_id         text not null,
  step_id            text,
  correlation_id     text,
  event_type         text not null,
  decision           text,
  tool               text,
  action             text,
  resource           text,
  risk_score         int,
  risk_tags          jsonb not null default '[]'::jsonb,
  matched_policy_ids jsonb not null default '[]'::jsonb,
  reason_code        text,
  reason_text        text,
  input_summary      text,
  output_summary     text,
  artifact_refs      jsonb not null default '[]'::jsonb,
  actor_type         text,
  actor_id           text,
  created_at         timestamptz not null default now()
);

create index idx_tool_events_session_time on tool_events(session_id, created_at);
create index idx_tool_events_corr on tool_events(correlation_id);
create index idx_tool_events_type_time on tool_events(event_type, created_at desc);
create index idx_tool_events_tool_action on tool_events(tool, action);
create index idx_tool_events_risk_tags on tool_events using gin (risk_tags);
