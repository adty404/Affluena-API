ALTER TABLE categories
	ADD CONSTRAINT categories_user_id_id_type_unique UNIQUE (user_id, id, type);

ALTER TABLE categories
	ADD CONSTRAINT categories_user_parent_type_fk
		FOREIGN KEY (user_id, parent_id, type) REFERENCES categories(user_id, id, type) ON DELETE RESTRICT;
