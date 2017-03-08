-- +migrate Up
ALTER table project_variable_audit RENAME TO project_variable_audit_old;

CREATE TABLE IF NOT EXISTS "project_variable_audit" (
  id BIGSERIAL PRIMARY KEY,
  project_id BIGINT,
  variable_id BIGINT,
  type TEXT,
  variable_before JSONB,
  variable_after JSONB,
  versionned TIMESTAMP WITH TIME ZONE,
  author TEXT
);

ALTER table application_variable_audit RENAME TO application_variable_audit_old;

CREATE TABLE IF NOT EXISTS "application_variable_audit" (
  id BIGSERIAL PRIMARY KEY,
  application_id BIGINT,
  variable_id BIGINT,
  type TEXT,
  variable_before JSONB,
  variable_after JSONB,
  versionned TIMESTAMP WITH TIME ZONE,
  author TEXT
);

ALTER table environment_variable_audit RENAME TO environment_variable_audit_old;

CREATE TABLE IF NOT EXISTS "environment_variable_audit" (
  id BIGSERIAL PRIMARY KEY,
  environment_id BIGINT,
  variable_name BIGINT,
  type TEXT,
  variable_before JSONB,
  variable_after JSONB,
  versionned TIMESTAMP WITH TIME ZONE,
  author TEXT
);
-- +migrate Down
DROP TABLE project_variable_audit;
DROP TABLE application_variable_audit;
DROP TABLE environment_variable_audit;
ALTER table project_variable_audit_old RENAME TO project_variable_audit;
ALTER table application_variable_audit_old RENAME TO application_variable_audit;
ALTER table environment_variable_audit_old RENAME TO environment_variable_audit;