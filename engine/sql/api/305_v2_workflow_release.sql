-- +migrate Up
CREATE TABLE v2_workflow_version
(
    "id"                uuid PRIMARY KEY,
    "version"           VARCHAR(255) NOT NULL,
    "project_key"       VARCHAR(255) NOT NULL,
    "vcs_server"        VARCHAR(255) NOT NULL,
    "repository"        VARCHAR(255) NOT NULL,
    "workflow_name"     VARCHAR(255) NOT NULL,
    "workflow_run_id"   uuid    NOT NULL,
    "username"          VARCHAR(255) NOT NULL,
    "user_id"           VARCHAR(255) NOT NULL,
    "sha"               VARCHAR(255) NOT NULL,
    "ref"               VARCHAR(255) NOT NULL,
    "type"              VARCHAR(255) NOT NULL,
    "file"              VARCHAR(255) NOT NULL,
    "created"           TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP
);
SELECT create_foreign_key_idx_cascade('FK_v2_workflow_version_project', 'v2_workflow_version', 'project', 'project_key', 'projectkey');
SELECT create_unique_index('v2_workflow_version', 'idx_unq_rv2_workflow_version', 'project_key,vcs_server,repository,workflow_name,version');

-- +migrate Down
DROP TABLE v2_workflow_version;