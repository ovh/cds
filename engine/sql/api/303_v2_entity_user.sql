-- +migrate Up
ALTER TABLE entity ADD COLUMN user_id VARCHAR(36) NULL;
ALTER TABLE entity ADD CONSTRAINT fk_entity_user FOREIGN KEY (user_id) REFERENCES authentified_user (id) ON DELETE SET NULL;

-- +migrate Down
ALTER TABLE entity DROP COLUMN user_id;
