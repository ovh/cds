-- +migrate Up
DROP TABLE warning;

CREATE TABLE warning (
  project_key VARCHAR(50) NOT NULL,
  application_name VARCHAR(50) NOT NULL default '',
  pipeline_name VARCHAR(50) NOT NULL default '',
  environment_name VARCHAR(50) NOT NULL default '',
  workflow_name VARCHAR(50) NOT NULL default '',
  type VARCHAR(100) NOT NULL,
  element VARCHAR(256) NOT NULL,
  created TIMESTAMP WITH TIME ZONE,
  message_params JSONB,
  PRIMARY KEY (type, element)
);

-- +migrate Down
DROP TABLE warning;
CREATE TABLE IF NOT EXISTS "warning" (id BIGSERIAL PRIMARY KEY, project_id BIGINT, app_id BIGINT, pip_id BIGINT, env_id BIGINT, action_id BIGINT, warning_id BIGINT, message_param JSONB);
ALTER TABLE warning ADD CONSTRAINT fk_application FOREIGN KEY (app_id) references application (id) ON delete cascade;
ALTER TABLE warning ADD CONSTRAINT fk_pipeline FOREIGN KEY (pip_id) references pipeline (id) ON delete cascade;
ALTER TABLE warning ADD CONSTRAINT fk_environment FOREIGN KEY (env_id) references environment (id) ON delete cascade;
ALTER TABLE warning ADD CONSTRAINT fk_action FOREIGN KEY (action_id) references action (id) ON delete cascade;
