-- +migrate Up
CREATE TABLE project_secret (
  id              VARCHAR(36)  PRIMARY KEY,
  project_key     VARCHAR(255) NOT NULL,
  name            TEXT NOT NULL,
  last_modified   TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  encrypted_value BYTEA
);
-- +migrate Down
DROP TABLE project_secret;

