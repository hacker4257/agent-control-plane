create table alerts (
  alert_id           text primary key,
  session_id         text not null references sessions(session_id),
  event_id           text,
  category           text not null,
  severity           text not null,
  status             text not null,
  title              text not null,
  summary            text,
  evidence_refs      jsonb not null default '[]'::jsonb,
  created_at         timestamptz not null default now(),
  updated_at         timestamptz not null default now()
);

create table artifacts (
  artifact_id        text primary key,
  session_id         text not null references sessions(session_id),
  event_id           text,
  artifact_type      text not null,
  title              text,
  storage_uri        text not null,
  content_summary    text,
  metadata           jsonb not null default '{}'::jsonb,
  created_at         timestamptz not null default now()
);

create table agents (
  agent_id           text primary key,
  display_name       text,
  environment        text,
  owner_team         text,
  permission_scope   jsonb not null default '{}'::jsonb,
  risk_profile       text,
  last_seen_at       timestamptz
);

create table tools (
  tool_id            text primary key,
  display_name       text,
  risk_level         text,
  policy_count       int not null default 0,
  blocked_24h        int not null default 0,
  approvals_24h      int not null default 0,
  last_used_at       timestamptz
);

create index idx_alerts_status_severity on alerts(status, severity, created_at desc);
create index idx_alerts_session on alerts(session_id);
create index idx_artifacts_session_time on artifacts(session_id, created_at desc);
create index idx_artifacts_type on artifacts(artifact_type);
