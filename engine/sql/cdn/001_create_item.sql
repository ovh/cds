-- +migrate Up

CREATE TABLE IF NOT EXISTS "item" (
  id VARCHAR(36) PRIMARY KEY, -- technical ID
  created TIMESTAMP WITH TIME ZONE, -- creation date
  last_modified TIMESTAMP WITH TIME ZONE, -- last modified date
  cipher_hash BYTEA, --
  api_ref JSONB, -- reference for cds PI
  api_ref_hash TEXT, -- hash of reference
  sig BYTEA,
  signer TEXT,
  status VARCHAR(64), -- status of item
  type VARCHAR(64), -- type of item
  size BIGINT,
  md5 TEXT,
  to_delete BOOLEAN
);
CREATE INDEX IDX_API_REF ON "item" USING GIN (api_ref);
select create_unique_index('item', 'IDX_ITEM_UNIQ_ITEM', 'api_ref_hash,type');
select create_index('item', 'IDX_ITEM_STATUS', 'status');

-- Index to get a log
CREATE INDEX IDX_LOG_ITEM ON item(type, (api_ref->>'job_id'), (api_ref->>'step_order'));
CREATE INDEX IDX_LOG_PROJECT_KEY ON item((api_ref->>'project_key'));

-- +migrate Down
DROP TABLE IF EXISTS "item";
