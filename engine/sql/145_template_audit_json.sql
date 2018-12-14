-- +migrate Up

ALTER TABLE workflow_template_audit ALTER COLUMN data_before TYPE JSONB USING data_after::JSONB;
ALTER TABLE workflow_template_audit ALTER COLUMN data_after TYPE JSONB USING data_after::JSONB;

-- +migrate Down

ALTER TABLE workflow_template_audit ALTER COLUMN data_before TYPE TEXT USING data_after::TEXT;
ALTER TABLE workflow_template_audit ALTER COLUMN data_after TYPE TEXT USING data_after::TEXT;
