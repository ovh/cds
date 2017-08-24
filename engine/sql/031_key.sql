-- +migrate Up
CREATE TABLE project_key (
  id BIGSERIAL,
  project_id BIGINT,
  name TEXT,
  type TEXT,
  public TEXT,
  private TEXT
);

SELECT create_foreign_key_idx_cascade('FK_PROJECT_KEY', 'project_key', 'project', 'project_id', 'id');
SELECT create_index('project_key', 'IDX_PROJECT_KEY_PROJECT_ID', 'project_id');
SELECT create_unique_index('project_key', 'IDX_PROJECT_KEY_NAME', 'project_id, name');

CREATE TABLE application_key (
  id BIGSERIAL,
  application_id BIGINT,
  name TEXT,
  type TEXT,
  public TEXT,
  private TEXT
);

SELECT create_foreign_key_idx_cascade('FK_APPLICATION_KEY', 'application_key', 'application', 'application_id', 'id');
SELECT create_index('application_key', 'IDX_APPLICATION_KEY_APPLICATION_ID', 'application_id');
SELECT create_unique_index('application_key', 'IDX_APPLICATION_KEY_NAME', 'application_id, name');

CREATE TABLE environment_key (
  id BIGSERIAL,
  environment_id BIGINT,
  name TEXT,
  type TEXT,
  public TEXT,
  private TEXT
);

SELECT create_foreign_key_idx_cascade('FK_ENVIRONMENT_KEY', 'environment_key', 'environment', 'environment_id', 'id');
SELECT create_index('environment_key', 'IDX_ENVIRONMENT_KEY_ENVIRONMENT_ID', 'environment_id');
SELECT create_unique_index('environment_key', 'IDX_ENVIRONMENT_KEY_NAME', 'environment_id, name');

-- +migrate Down
DROP TABLE project_key;
DROP TABLE application_key;
DROP TABLE environment_key;

