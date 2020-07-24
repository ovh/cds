-- +migrate Up
DROP TABLE warning;

-- +migrate Down
CREATE TABLE IF NOT EXISTS warning
(
    id BIGSERIAL PRIMARY KEY,
    project_key character varying(50),
    application_name character varying(50),
    pipeline_name character varying(50),
    environment_name character varying(50),
    workflow_name character varying(50),
    type character varying(100),
    element character varying(256),
    created timestamp with time zone,
    message_params jsonb,
    hash text,
    ignored boolean DEFAULT false
);