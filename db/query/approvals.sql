-- name: ListApprovalsByStatus :many
select *
from approvals
where status = $1
order by requested_at desc
limit $2 offset $3;

-- name: GetApprovalByID :one
select *
from approvals
where approval_id = $1;
