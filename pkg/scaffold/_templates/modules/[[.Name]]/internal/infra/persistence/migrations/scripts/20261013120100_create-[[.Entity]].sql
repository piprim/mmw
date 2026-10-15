-- +goose Up
-- +goose StatementBegin

-- The CHECK mirrors the Label value object: non-empty, at most 200 characters.
CREATE TABLE IF NOT EXISTS [[.Name]].[[.Entity]] (
    id BIGSERIAL PRIMARY KEY,
    label VARCHAR(200) NOT NULL CHECK (length(label) > 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);

COMMENT ON TABLE [[.Name]].[[.Entity]] IS 'First aggregate of the [[.Name]] module';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS [[.Name]].[[.Entity]];

-- +goose StatementEnd
