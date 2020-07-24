-- +migrate Up
ALTER TABLE worker_model
ADD COLUMN last_registration TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
ADD COLUMN need_registration BOOLEAN DEFAULT TRUE,
ADD COLUMN disabled BOOLEAN DEFAULT FALSE,
ADD COLUMN template JSONB,
ADD COLUMN communication TEXT DEFAULT 'http',
ADD COLUMN run_script TEXT DEFAULT '',
ADD COLUMN provision INT DEFAULT 0;

-- +migrate Down
ALTER TABLE worker_model
DROP COLUMN last_registration,
DROP COLUMN need_registration,
DROP COLUMN disabled,
DROP COLUMN template,
DROP COLUMN communication,
DROP COLUMN run_script,
DROP COLUMN provision;
