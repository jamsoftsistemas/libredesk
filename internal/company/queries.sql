-- name: get-companies
SELECT COUNT(*) OVER() AS total, id, created_at, updated_at, external_id, "name", "document", city, "system" FROM companies
WHERE ($1 = '' OR "name" ILIKE '%' || $1 || '%' OR "document" ILIKE '%' || $1 || '%' OR external_id ILIKE '%' || $1 || '%')
ORDER BY "name" LIMIT NULLIF($2, 0) OFFSET $3;

-- name: get-companies-compact
SELECT id, "name" FROM companies
WHERE ($1 = '' OR "name" ILIKE '%' || $1 || '%' OR external_id ILIKE '%' || $1 || '%')
ORDER BY "name" LIMIT NULLIF($2, 0) OFFSET $3;

-- name: get-companies-compact-by-ids
SELECT id, "name" FROM companies WHERE id = ANY($1) ORDER BY "name";

-- name: get-company
SELECT id, created_at, updated_at, external_id, "name", "document", city, "system" FROM companies WHERE id = $1;

-- name: get-company-by-external-id
SELECT id, created_at, updated_at, external_id, "name", "document", city, "system" FROM companies WHERE external_id = $1;

-- name: insert-company
INSERT INTO companies (external_id, "name", "document", city, "system") VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: update-company
UPDATE companies SET external_id = $2, "name" = $3, "document" = $4, city = $5, "system" = $6, updated_at = now() WHERE id = $1 RETURNING *;

-- name: delete-company
DELETE FROM companies WHERE id = $1;
