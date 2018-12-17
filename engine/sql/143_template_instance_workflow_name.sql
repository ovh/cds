-- +migrate Up

ALTER TABLE workflow_template_instance ADD COLUMN workflow_name VARCHAR(100) NOT NULL DEFAULT '';
UPDATE workflow_template_instance SET workflow_name = request->>'workflow_name';

-- +migrate Down

ALTER TABLE workflow_template_instance DROP COLUMN workflow_name;
