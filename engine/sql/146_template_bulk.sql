-- +migrate Up

CREATE TABLE IF NOT EXISTS "workflow_template_bulk" (
  id BIGSERIAL PRIMARY KEY,
  workflow_template_id BIGINT NOT NULL,
  operations JSONB,
);

SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TEMPLATE_BULK_TEMPLATE', 'workflow_template_bulk', 'workflow_template', 'workflow_template_id', 'id');

-- +migrate Down

DROP TABLE IF EXISTS workflow_template_bulk;
