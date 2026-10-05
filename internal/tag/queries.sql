-- name: get-all-tags
select
    id,
    created_at,
    updated_at,
    name,
    visibility,
    team_id,
    inbox_id
from
    tags
where
    ($1 = '' or name ilike '%' || $1 || '%')
order by
    name
limit NULLIF($2, 0) offset $3;

-- name: get-tags-by-ids
select
    id,
    created_at,
    updated_at,
    name,
    visibility,
    team_id,
    inbox_id
from
    tags
where
    id = ANY($1)
order by
    name;

-- name: get-scoped-tags
select
    id,
    created_at,
    updated_at,
    name,
    visibility,
    team_id,
    inbox_id
from
    tags
where
    ($1 = '' or name ilike '%' || $1 || '%')
    and (
        visibility = 'all'
        or (visibility = 'team' and team_id = $2)
        or (visibility = 'inbox' and inbox_id = $3)
    )
order by
    name;

-- name: insert-tag
INSERT into
    tags (name, visibility, team_id, inbox_id)
values
    ($1, $2, $3, $4)
RETURNING *;

-- name: delete-tag
DELETE from
    tags
where
    id = $1;

-- name: update-tag
UPDATE
    tags
set
    name = $2,
    visibility = $3,
    team_id = $4,
    inbox_id = $5,
    updated_at = now()
where
    id = $1
RETURNING *;
