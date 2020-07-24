-- +migrate Up

ALTER TABLE workflow_template_audit DROP COLUMN IF EXISTS data_type;

-- +migrate Down

ALTER TABLE workflow_template_audit ADD COLUMN data_type VARCHAR(100);
UPDATE workflow_template_audit SET data_type = 'json';
