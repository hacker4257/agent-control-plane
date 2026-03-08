-- name: ListAgents :many
select * from agents order by last_seen_at desc nulls last;

-- name: ListTools :many
select * from tools order by tool_id asc;
