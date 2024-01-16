-- +migrate Up
CREATE TABLE project_notification (
  id              uuid  PRIMARY KEY,
  project_key     VARCHAR(255) NOT NULL,
  name            VARCHAR(255) NOT NULL,
  last_modified   TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
  webhook_url     TEXT NOT NULL,
  filters         JSONB NOT NULL,
  auth            BYTEA,
  "sig"           BYTEA,
  "signer"        TEXT
);

SELECT create_foreign_key_idx_cascade('fk_project_notification', 'project_notification', 'project', 'project_key', 'projectkey');
SELECT create_unique_index('project_notification', 'IDX_project_notification_unq', 'project_key,name');

-- +migrate Down
DROP TABLE project_notification;