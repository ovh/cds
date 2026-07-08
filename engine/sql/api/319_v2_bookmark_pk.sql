-- +migrate Up
ALTER TABLE entity_favorite DROP CONSTRAINT entity_favorite_pkey;
ALTER TABLE entity_favorite ADD PRIMARY KEY (authentified_user_id, project_repository_id, type, name);

-- +migrate Down
ALTER TABLE entity_favorite DROP CONSTRAINT entity_favorite_pkey;
ALTER TABLE entity_favorite ADD PRIMARY KEY (authentified_user_id, type, name);
