-- +migrate Up

-- Create new table for user registrations
CREATE TABLE IF NOT EXISTS "index" (
  id VARCHAR(36) PRIMARY KEY,
  created TIMESTAMP WITH TIME ZONE,
  name VARCHAR(255) NOT NULL,
  sig BYTEA,
  signer TEXT
);

-- +migrate Down
DROP TABLE IF EXISTS "index";
