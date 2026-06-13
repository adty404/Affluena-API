ALTER TABLE tags
	ADD CONSTRAINT tags_user_id_id_unique UNIQUE (user_id, id);

ALTER TABLE transaction_tags
	ADD COLUMN user_id uuid;

UPDATE transaction_tags tt
SET user_id = t.user_id
FROM transactions t
WHERE tt.transaction_id = t.id;

ALTER TABLE transaction_tags
	ALTER COLUMN user_id SET NOT NULL,
	ADD CONSTRAINT transaction_tags_user_transaction_fk
		FOREIGN KEY (user_id, transaction_id) REFERENCES transactions(user_id, id) ON DELETE CASCADE,
	ADD CONSTRAINT transaction_tags_user_tag_fk
		FOREIGN KEY (user_id, tag_id) REFERENCES tags(user_id, id) ON DELETE CASCADE;

CREATE INDEX transaction_tags_user_tag_id_idx ON transaction_tags(user_id, tag_id);
