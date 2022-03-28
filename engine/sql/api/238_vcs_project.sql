-- +migrate Up
CREATE TABLE vcs_project
(
    "id"            uuid PRIMARY KEY,
    "name"          VARCHAR(255) NOT NULL,
    "type"          VARCHAR(64) NOT NULL,    
    "project_id"    BIGINT,
    "description"   VARCHAR(255),
    "url"           VARCHAR(255) NOT NULL,
    "auth"          BYTEA,
    "created"       TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,    
    "last_modified" TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    "created_by"    VARCHAR(255) NOT NULL,
    "sig"           BYTEA,
    "signer"        TEXT
);
SELECT create_unique_index('vcs_project', 'idx_unq_name_project_id', 'name,project_id');
SELECT create_foreign_key_idx_cascade('fk_vcs_project_project', 'vcs_project', 'project', 'project_id', 'id');

ALTER TABLE "rbac" RENAME COLUMN "uuid" TO "id";
ALTER TABLE "rbac_project" RENAME COLUMN "rbac_uuid" TO "rbac_id";
ALTER TABLE "rbac_global" RENAME COLUMN "rbac_uuid" TO "rbac_id";

-- +migrate Down
DROP TABLE vcs_project;
ALTER TABLE "rbac" RENAME COLUMN "id" TO "uuid";
ALTER TABLE "rbac_project" RENAME COLUMN "rbac_id" TO "rbac_uuid";
ALTER TABLE "rbac_global" RENAME COLUMN "rbac_id" TO "rbac_uuid";