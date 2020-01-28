-- +migrate Up
ALTER TABLE "group" ADD COLUMN sig BYTEA;
ALTER TABLE "group" ADD COLUMN signer TEXT;

CREATE TABLE "group_authentified_user" (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    authentified_user_id VARCHAR(36) NOT NULL,
    group_admin BOOLEAN,
    sig BYTEA,
    signer TEXT
);

SELECT create_foreign_key_idx_cascade('FK_GROUP_AUTHENTIFIED_USER_USER', 'group_authentified_user', 'group', 'group_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_GROUP_AUTHENTIFIED_USER_AUTHENTIFIED_USER', 'group_authentified_user', 'authentified_user', 'authentified_user_id', 'id');

ALTER TABLE "project_group" ADD COLUMN sig BYTEA;
ALTER TABLE "project_group" ADD COLUMN signer TEXT;

-- +migrate Down



