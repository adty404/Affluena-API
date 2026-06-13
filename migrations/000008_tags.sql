CREATE TABLE tags (
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	name text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	UNIQUE (user_id, name)
);

CREATE TABLE transaction_tags (
	transaction_id uuid NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
	tag_id uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
	PRIMARY KEY (transaction_id, tag_id)
);

CREATE INDEX transaction_tags_tag_id_idx ON transaction_tags(tag_id);
