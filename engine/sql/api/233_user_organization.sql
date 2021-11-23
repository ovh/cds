-- +migrate Up
CREATE TABLE IF NOT EXISTS "authentified_user_organization" (
  id BIGSERIAL PRIMARY KEY,
  authentified_user_id VARCHAR(36) NOT NULL,
  organization VARCHAR(100) NOT NULL,
  sig BYTEA,
  signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_AUTHENTIFIED_USER_ORGANIZATION_AUTHENTIFIED_USER', 'authentified_user_organization', 'authentified_user', 'authentified_user_id', 'id');

CREATE TABLE IF NOT EXISTS "group_organization" (
  id BIGSERIAL PRIMARY KEY,
  group_id BIGINT NOT NULL,
  organization VARCHAR(100) NOT NULL,
  sig BYTEA,
  signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_GROUP_ORGANIZATION_GROUP', 'group_organization', 'group', 'group_id', 'id');

-- +migrate Down
DROP TABLE IF EXISTS "authentified_user_organization";
DROP TABLE IF EXISTS "group_organization";
