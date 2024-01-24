-- +migrate Up
CREATE TABLE "project_variable_set" (
    id              uuid  PRIMARY KEY,  
    project_key     VARCHAR(255) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    created         TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "sig"           BYTEA,
    "signer"        TEXT
);
SELECT create_unique_index('project_variable_set', 'IDX_project_variable_set_unq', 'project_key,name');
SELECT create_foreign_key_idx_cascade('fk_project_variable_set', 'project_variable_set', 'project', 'project_key', 'projectkey');

CREATE TABLE "project_variable_set_text" (
    id                          uuid  PRIMARY KEY,  
    project_variable_set_id     uuid NOT NULL,
    name                        VARCHAR(255) NOT NULL,
    last_modified               TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    value                       TEXT,
    "sig"                       BYTEA,
    "signer"                    TEXT
);
SELECT create_unique_index('project_variable_set_text', 'IDX_project_variable_set_text_unq', 'project_variable_set_id,name');
SELECT create_foreign_key_idx_cascade('fk_project_variable_set_text', 'project_variable_set_text', 'project_variable_set', 'project_variable_set_id', 'id');

CREATE TABLE "project_variable_set_secret" (
    id                          uuid  PRIMARY KEY,  
    project_variable_set_id     uuid NOT NULL,
    name                        VARCHAR(255) NOT NULL,
    last_modified               TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    value                       BYTEA,
    "sig"                       BYTEA,
    "signer"                    TEXT
);
SELECT create_unique_index('project_variable_set_secret', 'IDX_project_variable_set_secret_unq', 'project_variable_set_id,name');
SELECT create_foreign_key_idx_cascade('fk_project_variable_set_secret', 'project_variable_set_secret', 'project_variable_set', 'project_variable_set_id', 'id');

-- +migrate Down
DROP TABLE project_variable_set_secret;
DROP TABLE project_variable_set_text;
DROP TABLE project_variable_set;

