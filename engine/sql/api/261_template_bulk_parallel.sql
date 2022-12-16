
-- +migrate Up
ALTER TABLE workflow_template_bulk ADD COLUMN parallel BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE workflow_template_bulk ADD COLUMN auth_consumer_id VARCHAR(36);
ALTER TABLE workflow_template_bulk ADD COLUMN status BIGINT NOT NULL DEFAULT 2;

-- +migrate Down
ALTER TABLE workflow_template_bulk DROP COLUMN parallel;
ALTER TABLE workflow_template_bulk DROP COLUMN auth_consumer_id;
ALTER TABLE workflow_template_bulk DROP COLUMN status;
