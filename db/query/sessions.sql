-- name: ListSessions :many
select *
from sessions
order by last_event_at desc nulls last
limit $1 offset $2;

-- name: GetSessionByID :one
select *
from sessions
where session_id = $1;
