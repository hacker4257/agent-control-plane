-- name: ListAlerts :many
select *
from alerts
order by created_at desc
limit $1 offset $2;

-- name: GetAlertByID :one
select *
from alerts
where alert_id = $1;
