create table approvals (
  approval_id        text primary key,
  session_id         text not null references sessions(session_id),
  step_id            text,
  event_id           text,
  status             text not null,
  action             text not null,
  tool               text not null,
  resource           text not null,
  objective          text,
  trigger_reason     text not null,
  risk_tags          jsonb not null default '[]'::jsonb,
  potential_impact   text,
  suggested_safe_alt text,
  requested_at       timestamptz not null default now(),
  decided_at         timestamptz,
  approver_id        text,
  decision_comment   text
);

create table policy_rules (
  policy_id          text primary key,
  name               text not null unique,
  description        text,
  scope_agent        text,
  scope_tool         text,
  scope_environment  text,
  scope_resource_pat text,
  condition_expr     jsonb not null,
  decision           text not null,
  priority           int not null default 100,
  enabled            boolean not null default true,
  created_by         text,
  created_at         timestamptz not null default now(),
  updated_at         timestamptz not null default now()
);

create index idx_approvals_status_time on approvals(status, requested_at desc);
create index idx_approvals_session on approvals(session_id);
create index idx_policy_rules_enabled_pri on policy_rules(enabled, priority);
create index idx_policy_rules_scope_tool on policy_rules(scope_tool);
