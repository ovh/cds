-- +migrate Up

CREATE TABLE IF NOT EXISTS "index" (
  id VARCHAR(36) PRIMARY KEY, -- technical ID
  created TIMESTAMP WITH TIME ZONE, -- creation date
  last_modified TIMESTAMP WITH TIME ZONE, -- last modified date
  cipher_hash BYTEA, --
  api_ref JSONB, -- reference for cds PI
  api_ref_hash TEXT, -- hash of reference
  sig BYTEA,
  signer TEXT,
  status VARCHAR(64), -- status of item
  type VARCHAR(64) -- type of item
);
CREATE INDEX api_ref_index ON "index" USING GIN (api_ref);
select create_unique_index('index', 'IDX_UNIQ_ITEM', 'api_ref_hash,type');

-- Index to get a log
CREATE INDEX IDX_LOG_ITEM ON index(type, (api_ref->>'job_id'), (api_ref->>'step_order'));

-- +migrate Down
DROP TABLE IF EXISTS "index";
