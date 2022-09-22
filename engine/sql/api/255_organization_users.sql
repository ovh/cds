-- +migrate Up
ALTER TABLE authentified_user_organization RENAME TO authentified_user_organization_old;

CREATE TABLE IF NOT EXISTS "authentified_user_organization" (
    id uuid PRIMARY KEY,
    authentified_user_id VARCHAR(36) NOT NULL,
    organization_id uuid NOT NULL,
    sig BYTEA,
    signer TEXT
    );
DROP INDEX IF EXISTS IDX_FK_AUTHENTIFIED_USER_ORGANIZATION_AUTHENTIFIED_USER;
SELECT create_foreign_key_idx_cascade('FK_AUTHENTIFIED_USER_ORGANIZATION_AUTHENTIFIED_USER', 'authentified_user_organization', 'authentified_user', 'authentified_user_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_AUTHENTIFIED_USER_ORGANIZATION_ORGANIZATION', 'authentified_user_organization', 'organization', 'organization_id', 'id');

ALTER TABLE group_organization RENAME TO group_organization_old;
CREATE TABLE IF NOT EXISTS "group_organization" (
    id uuid PRIMARY KEY,
    group_id BIGINT NOT NULL,
    organization_id uuid NOT NULL,
    sig BYTEA,
    signer TEXT
);
DROP INDEX IF EXISTS IDX_FK_GROUP_ORGANIZATION_GROUP;
SELECT create_foreign_key_idx_cascade('FK_GROUP_ORGANIZATION_GROUP', 'group_organization', 'group', 'group_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_GROUP_ORGANIZATION_ORGANIZATION', 'group_organization', 'organization', 'organization_id', 'id');


-- +migrate Down
DROP TABLE authentified_user_organization;
ALTER TABLE authentified_user_organization_old RENAME TO authentified_user_organization;

DROP TABLE group_organization;
ALTER TABLE group_organization_old RENAME TO group_organization;

DROP INDEX IF EXISTS IDX_FK_AUTHENTIFIED_USER_ORGANIZATION_AUTHENTIFIED_USER;
SELECT create_foreign_key_idx_cascade('FK_AUTHENTIFIED_USER_ORGANIZATION_AUTHENTIFIED_USER', 'authentified_user_organization', 'authentified_user', 'authentified_user_id', 'id');

DROP INDEX IF EXISTS IDX_FK_GROUP_ORGANIZATION_GROUP;
SELECT create_foreign_key_idx_cascade('FK_GROUP_ORGANIZATION_GROUP', 'group_organization', 'group', 'group_id', 'id');
