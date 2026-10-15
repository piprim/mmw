-- Queries compiled by sqlc into this package (see sqlc.yaml at the module root).
-- sqlc needs dao/schema.sql: run `mise run db:schema:dump`, then
-- `mise run sqlc:generate`. Use sqlc for selects whose result fits no boa
-- model; simple row-shaped queries stay in boa.

-- name: [[.Entity | pascal]]GetByID :one
SELECT id, label, created_at, updated_at
FROM [[.Name]].[[.Entity]]
WHERE id = $1;
