-- +goose Up
-- +goose StatementBegin
ALTER TABLE categories
ADD COLUMN parent_id UUID REFERENCES categories(id) ON DELETE RESTRICT;

CREATE INDEX idx_categories_parent_id ON categories(parent_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_categories_parent_id;

ALTER TABLE categories
DROP COLUMN parent_id;
-- +goose StatementEnd
