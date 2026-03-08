-- name: ListPolicies :many
select *
from policy_rules
where ($1::bool is null or enabled = $1)
order by priority asc, created_at desc
limit $2 offset $3;

-- name: GetPolicyByID :one
select *
from policy_rules
where policy_id = $1;
