-- +goose Up
-- +goose StatementBegin

INSERT INTO [[.Name]].[[.Entity]] (label, created_at, updated_at)
VALUES
    ('Premier [[.Entity]]', NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day'),
    ('Second [[.Entity]]', NOW() - INTERVAL '1 day', NOW());

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

TRUNCATE TABLE [[.Name]].[[.Entity]];

-- +goose StatementEnd
