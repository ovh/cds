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

CREATE TABLE project_platform (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(100),
  project_id BIGINT,
  platform_model_id BIGINT,
  config JSONB
);



-- +migrate Down
DROP TABLE platform_model;
DROP TABLE project_platform;