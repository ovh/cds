-- +migrate Up
CREATE TABLE platform_model (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(50),
  author VARCHAR(50),
  identifier VARCHAR(150),
  icon VARCHAR(20),
  default_config JSONB,
  disabled BOOLEAN,
  hook BOOLEAN,
  file_storage BOOLEAN,
  block_storage BOOLEAN,
  deployment BOOLEAN,
  compute BOOLEAN
);

select create_unique_index('platform_model', 'IDX_PLATFORM_MODEL_NAME', 'name');

CREATE TABLE project_platform (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(100),
  project_id BIGINT,
  platform_model_id BIGINT,
  config JSONB
);


SELECT create_foreign_key_idx_cascade('fk_project_platform', 'project_platform', 'project', 'project_id', 'id');
select create_unique_index('project_platform', 'IDX_PROJECT_PLATFORM_NAME', 'project_id,name');
select create_index('project_platform', 'IDX_PROJECT_PLATFORM', 'project_id,name');

-- +migrate Down
DROP TABLE platform_model;
DROP TABLE project_platform;