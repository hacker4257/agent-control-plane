create table sessions (
  session_id         text primary key,
  objective          text not null,
  agent_id           text not null,
  user_id            text,
  environment        text not null,
  status             text not null,
  started_at         timestamptz not null,
  ended_at           timestamptz,
  risk_score         int not null default 0,
  approvals_count    int not null default 0,
  blocked_count      int not null default 0,
  touched_resources  jsonb not null default '[]'::jsonb,
  last_event_at      timestamptz,
  updated_at         timestamptz not null default now()
);

create table steps (
  step_id            text primary key,
  session_id         text not null references sessions(session_id),
  sequence_no        int not null,
  title              text,
  status             text not null,
  started_at         timestamptz not null default now(),
  ended_at           timestamptz
);

create index idx_sessions_status on sessions(status);
create index idx_sessions_risk on sessions(risk_score desc);
create index idx_sessions_agent on sessions(agent_id);
create index idx_sessions_user on sessions(user_id);
create index idx_sessions_env on sessions(environment);
create index idx_sessions_last_event on sessions(last_event_at desc);
create index idx_steps_session_seq on steps(session_id, sequence_no);
