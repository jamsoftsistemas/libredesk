-- name: get-all-tags
select
    t.id,
    t.created_at,
    t.updated_at,
    t.name,
    coalesce((select array_agg(team_id order by team_id) from tag_teams where tag_id = t.id), '{}') as team_ids,
    coalesce((select array_agg(inbox_id order by inbox_id) from tag_inboxes where tag_id = t.id), '{}') as inbox_ids
from
    tags t
where
    ($1 = '' or t.name ilike '%' || $1 || '%')
order by
    t.name
limit NULLIF($2, 0) offset $3;

-- name: get-tags-by-ids
select
    t.id,
    t.created_at,
    t.updated_at,
    t.name,
    coalesce((select array_agg(team_id order by team_id) from tag_teams where tag_id = t.id), '{}') as team_ids,
    coalesce((select array_agg(inbox_id order by inbox_id) from tag_inboxes where tag_id = t.id), '{}') as inbox_ids
from
    tags t
where
    t.id = ANY($1)
order by
    t.name;

-- name: get-scoped-tags
select
    t.id,
    t.created_at,
    t.updated_at,
    t.name,
    coalesce((select array_agg(team_id order by team_id) from tag_teams where tag_id = t.id), '{}') as team_ids,
    coalesce((select array_agg(inbox_id order by inbox_id) from tag_inboxes where tag_id = t.id), '{}') as inbox_ids
from
    tags t
where
    ($1 = '' or t.name ilike '%' || $1 || '%')
    and (
        -- Global: no team/inbox scoping rows at all.
        (
            not exists (select 1 from tag_teams where tag_id = t.id)
            and not exists (select 1 from tag_inboxes where tag_id = t.id)
        )
        or exists (select 1 from tag_teams where tag_id = t.id and team_id = $2)
        or exists (select 1 from tag_inboxes where tag_id = t.id and inbox_id = $3)
    )
order by
    t.name;

-- name: insert-tag
INSERT into
    tags (name)
values
    ($1)
RETURNING id, created_at, updated_at, name;

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
    updated_at = now()
where
    id = $1
RETURNING id, created_at, updated_at, name;

-- name: set-tag-teams
WITH inserted AS (
    INSERT INTO tag_teams (tag_id, team_id)
    SELECT $1, team_id FROM unnest($2::bigint[]) AS team_id
    ON CONFLICT (tag_id, team_id) DO NOTHING
)
DELETE FROM tag_teams
WHERE tag_id = $1
AND team_id NOT IN (SELECT team_id FROM unnest($2::bigint[]) AS team_id);

-- name: set-tag-inboxes
WITH inserted AS (
    INSERT INTO tag_inboxes (tag_id, inbox_id)
    SELECT $1, inbox_id FROM unnest($2::int[]) AS inbox_id
    ON CONFLICT (tag_id, inbox_id) DO NOTHING
)
DELETE FROM tag_inboxes
WHERE tag_id = $1
AND inbox_id NOT IN (SELECT inbox_id FROM unnest($2::int[]) AS inbox_id);
