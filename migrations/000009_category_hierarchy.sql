ALTER TABLE categories
	ADD COLUMN parent_id uuid REFERENCES categories(id) ON DELETE RESTRICT;

CREATE INDEX idx_categories_parent_id ON categories(parent_id);
