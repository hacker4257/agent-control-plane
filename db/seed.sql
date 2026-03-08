-- Seed sessions (20)
insert into sessions (
  session_id, objective, agent_id, user_id, environment, status,
  started_at, ended_at, risk_score, approvals_count, blocked_count,
  touched_resources, last_event_at, updated_at
)
select
  'sess_seed_' || lpad(i::text, 3, '0'),
  case when i % 4 = 0 then 'Fix prod auth bug' when i % 4 = 1 then 'Run payment flow checks' when i % 4 = 2 then 'Generate customer response' else 'Cleanup temp logs' end,
  case when i % 4 = 0 then 'coding-agent-prod' when i % 4 = 1 then 'browser-agent-ap' when i % 4 = 2 then 'mail-agent' else 'ops-agent' end,
  'user_' || ((i % 5) + 1),
  case when i % 3 = 0 then 'prod' else 'staging' end,
  case when i % 5 = 0 then 'blocked' when i % 5 = 1 then 'approval_pending' when i % 5 = 2 then 'completed' else 'running' end,
  now() - ((i + 2) || ' hours')::interval,
  case when i % 5 in (0,2) then now() - ((i + 1) || ' hours')::interval else null end,
  (30 + (i * 3)) % 100,
  (i % 3),
  (i % 2),
  to_jsonb(array['repo:org/api-service', 'env:prod']),
  now() - (i || ' minutes')::interval,
  now() - (i || ' minutes')::interval
from generate_series(1, 20) as g(i)
on conflict (session_id) do nothing;

-- Seed tool events (100)
insert into tool_events (
  event_id, session_id, step_id, correlation_id, event_type, decision,
  tool, action, resource, risk_score, risk_tags, matched_policy_ids,
  reason_code, reason_text, input_summary, output_summary, artifact_refs,
  actor_type, actor_id, created_at
)
select
  'evt_seed_' || lpad(i::text, 4, '0'),
  'sess_seed_' || lpad((((i - 1) % 20) + 1)::text, 3, '0'),
  'step_' || (((i - 1) % 6) + 1),
  'corr_seed_' || (((i - 1) % 30) + 1),
  case when i % 7 = 0 then 'blocked'
       when i % 7 = 1 then 'approval_requested'
       when i % 7 = 2 then 'failed'
       else 'executed' end,
  case when i % 7 = 0 then 'BLOCK'
       when i % 7 = 1 then 'REQUIRE_APPROVAL'
       else 'ALLOW' end,
  case when i % 4 = 0 then 'github' when i % 4 = 1 then 'shell' when i % 4 = 2 then 'browser' else 'file' end,
  case when i % 4 = 0 then 'push' when i % 4 = 1 then 'exec' when i % 4 = 2 then 'submit' else 'write' end,
  case when i % 4 = 0 then 'repo:org/api-service/branch:main'
       when i % 4 = 1 then 'host:prod'
       when i % 4 = 2 then 'https://payment.example.com/checkout'
       else '/srv/app/config.yml' end,
  (40 + i) % 100,
  case when i % 7 = 0 then '["destructive_action"]'::jsonb
       when i % 7 = 1 then '["repo_write"]'::jsonb
       else '["normal"]'::jsonb end,
  case when i % 7 in (0,1) then '["pol_seed_001"]'::jsonb else '[]'::jsonb end,
  case when i % 7 = 0 then 'SHELL_DANGEROUS_COMMAND'
       when i % 7 = 1 then 'PROTECTED_BRANCH_WRITE'
       else 'DEFAULT_ALLOW' end,
  case when i % 7 = 0 then 'Blocked dangerous shell command'
       when i % 7 = 1 then 'Main branch write requires approval'
       else 'Executed normally' end,
  'seed input ' || i,
  'seed output ' || i,
  '[]'::jsonb,
  'agent',
  case when i % 4 = 0 then 'coding-agent-prod' when i % 4 = 1 then 'ops-agent' when i % 4 = 2 then 'browser-agent-ap' else 'mail-agent' end,
  now() - (i || ' minutes')::interval
from generate_series(1, 100) as g(i)
on conflict (event_id) do nothing;

-- Seed policy rules (4)
insert into policy_rules (
  policy_id, name, description, scope_tool, scope_environment, condition_expr, decision, priority, enabled, created_by
)
values
  (
    'pol_seed_001',
    'Protect main branch in prod',
    'Require approval for GitHub push to main in prod',
    'github',
    'prod',
    '{"action":"push","resource_contains":"branch:main"}'::jsonb,
    'REQUIRE_APPROVAL',
    10,
    true,
    'seed'
  ),
  (
    'pol_seed_002',
    'Block dangerous shell',
    'Block dangerous shell command patterns',
    'shell',
    null,
    '{"command_patterns":["rm -rf","curl|sh"]}'::jsonb,
    'BLOCK',
    5,
    true,
    'seed'
  ),
  (
    'pol_seed_003',
    'Block payment submit',
    'Block browser submit on payment pages',
    'browser',
    'prod',
    '{"action":"submit","resource_contains":"payment"}'::jsonb,
    'BLOCK',
    15,
    true,
    'seed'
  ),
  (
    'pol_seed_004',
    'Default allow',
    'Allow non-matching operations',
    null,
    null,
    '{}'::jsonb,
    'ALLOW',
    100,
    true,
    'seed'
  )
on conflict (policy_id) do nothing;

-- Seed approvals (10 pending + 10 decided)
insert into approvals (
  approval_id, session_id, step_id, event_id, status, action, tool, resource,
  objective, trigger_reason, risk_tags, potential_impact, suggested_safe_alt,
  requested_at, decided_at, approver_id, decision_comment
)
select
  'appr_seed_' || lpad(i::text, 3, '0'),
  'sess_seed_' || lpad((((i - 1) % 20) + 1)::text, 3, '0'),
  'step_' || (((i - 1) % 6) + 1),
  'evt_seed_' || lpad(i::text, 4, '0'),
  case when i <= 10 then 'pending' when i % 2 = 0 then 'approved' else 'rejected' end,
  case when i % 3 = 0 then 'push' when i % 3 = 1 then 'exec' else 'submit' end,
  case when i % 3 = 0 then 'github' when i % 3 = 1 then 'shell' else 'browser' end,
  case when i % 3 = 0 then 'repo:org/api-service/branch:main'
       when i % 3 = 1 then 'host:prod'
       else 'https://payment.example.com/checkout' end,
  case when i % 2 = 0 then 'Fix prod auth bug' else 'Run payment flow checks' end,
  case when i % 3 = 0 then 'Main branch write requires approval'
       when i % 3 = 1 then 'Dangerous shell command pattern detected'
       else 'Browser submit on payment resource is blocked' end,
  case when i % 3 = 0 then '["repo_write","protected_branch"]'::jsonb
       when i % 3 = 1 then '["destructive_action","shell"]'::jsonb
       else '["financial_action","browser"]'::jsonb end,
  'Potential impact from sensitive operation',
  'Use staging or create a PR with review',
  now() - ((i + 5) || ' minutes')::interval,
  case when i <= 10 then null else now() - ((i + 1) || ' minutes')::interval end,
  case when i <= 10 then null else 'approver_' || ((i % 3) + 1) end,
  case when i <= 10 then null when i % 2 = 0 then 'Approved after review' else 'Rejected by policy owner' end
from generate_series(1, 20) as g(i)
on conflict (approval_id) do nothing;
