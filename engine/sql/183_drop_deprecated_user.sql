-- +migrate Up
ALTER TABLE "group" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "group" ADD COLUMN IF NOT EXISTS signer TEXT;

CREATE TABLE IF NOT EXISTS "group_authentified_user" (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    authentified_user_id VARCHAR(36) NOT NULL,
    group_admin BOOLEAN,
    sig BYTEA,
    signer TEXT
);
SELECT create_foreign_key_idx_cascade('FK_GROUP_AUTHENTIFIED_USER_USER', 'group_authentified_user', 'group', 'group_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_GROUP_AUTHENTIFIED_USER_AUTHENTIFIED_USER', 'group_authentified_user', 'authentified_user', 'authentified_user_id', 'id');

ALTER TABLE "project_group" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "project_group" ADD COLUMN IF NOT EXISTS signer TEXT;

ALTER TABLE "broadcast_read" ADD COLUMN authentified_user_id VARCHAR(36);
SELECT create_foreign_key_idx_cascade('FK_BROADCAST_READ_AUTHENTIFIED_USER', 'broadcast_read', 'authentified_user', 'authentified_user_id', 'id');
ALTER TABLE "broadcast_read" DROP CONSTRAINT broadcast_read_pkey;
ALTER TABLE "broadcast_read" ALTER COLUMN user_id drop not null;
WITH sub AS (
    SELECT authentified_user_id, user_id
    FROM authentified_user_migration
)
UPDATE "broadcast_read" 
SET authentified_user_id = sub.authentified_user_id
FROM sub
WHERE broadcast_read.user_id = sub.user_id;
ALTER TABLE "broadcast_read" ALTER COLUMN authentified_user_id SET NOT NULL;
ALTER TABLE "broadcast_read" ADD PRIMARY KEY (broadcast_id, authentified_user_id);


ALTER TABLE "project_favorite" ADD COLUMN authentified_user_id VARCHAR(36);
SELECT create_foreign_key_idx_cascade('FK_PROJECT_FAVORITE_AUTHENTIFIED_USER', 'project_favorite', 'authentified_user', 'authentified_user_id', 'id');
ALTER TABLE "project_favorite" DROP CONSTRAINT project_favorite_pkey;
ALTER TABLE "project_favorite" ALTER COLUMN user_id drop not null;
WITH sub AS (
    SELECT authentified_user_id, user_id
    FROM authentified_user_migration
)
UPDATE "project_favorite" 
SET authentified_user_id = sub.authentified_user_id
FROM sub
WHERE project_favorite.user_id = sub.user_id;
ALTER TABLE "project_favorite" ALTER COLUMN authentified_user_id SET NOT NULL;
ALTER TABLE "project_favorite" ADD PRIMARY KEY (project_id, authentified_user_id);


ALTER TABLE "workflow_favorite" ADD COLUMN authentified_user_id VARCHAR(36);
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_FAVORITE_AUTHENTIFIED_USER', 'workflow_favorite', 'authentified_user', 'authentified_user_id', 'id');
ALTER TABLE "workflow_favorite" DROP CONSTRAINT workflow_favorite_pkey;
ALTER TABLE "workflow_favorite" ALTER COLUMN user_id drop not null;
WITH sub AS (
    SELECT authentified_user_id, user_id
    FROM authentified_user_migration
)
UPDATE "workflow_favorite" 
SET authentified_user_id = sub.authentified_user_id
FROM sub
WHERE workflow_favorite.user_id = sub.user_id;
ALTER TABLE "workflow_favorite" ALTER COLUMN authentified_user_id SET NOT NULL;
ALTER TABLE "workflow_favorite" ADD PRIMARY KEY (workflow_id, authentified_user_id);


ALTER TABLE "user_timeline" ADD COLUMN authentified_user_id VARCHAR(36);
SELECT create_foreign_key_idx_cascade('FK_USER_TIMELINE_AUTHENTIFIED_USER', 'user_timeline', 'authentified_user', 'authentified_user_id', 'id');
ALTER TABLE "user_timeline" DROP CONSTRAINT user_timeline_pkey;
ALTER TABLE "user_timeline" ALTER COLUMN user_id drop not null;
WITH sub AS (
    SELECT authentified_user_id, user_id
    FROM authentified_user_migration
)
UPDATE "user_timeline" 
SET authentified_user_id = sub.authentified_user_id
FROM sub
WHERE user_timeline.user_id = sub.user_id;
ALTER TABLE "user_timeline" ALTER COLUMN authentified_user_id SET NOT NULL;
ALTER TABLE "user_timeline" ADD PRIMARY KEY (authentified_user_id);


TRUNCATE TABLE  "workflow_template_bulk";
ALTER TABLE "workflow_template_bulk" ADD COLUMN authentified_user_id VARCHAR(36) NOT NULL;
SELECT create_foreign_key_idx_cascade('FK_WORKFLOW_TEMPLATE_BULK_AUTHENTIFIED_USER', 'workflow_template_bulk', 'authentified_user', 'authentified_user_id', 'id');
ALTER TABLE "workflow_template_bulk" ALTER COLUMN user_id drop not null;


UPDATE worker_model
SET created_by = json_build_object(
    'username', created_by->>'username',
    'fullname', created_by->>'fullname',
    'email', created_by->>'email');

-- +migrate Down

ALTER TABLE "group" DROP COLUMN sig;
ALTER TABLE "group" DROP COLUMN signer;
DROP TABLE "group_authentified_user";

ALTER TABLE "project_group" DROP COLUMN sig;
ALTER TABLE "project_group" DROP COLUMN signer;

ALTER TABLE "broadcast_read" DROP CONSTRAINT broadcast_read_pkey;
ALTER TABLE "broadcast_read" ADD PRIMARY KEY (broadcast_id, user_id);
ALTER TABLE "broadcast_read" DROP COLUMN authentified_user_id;

ALTER TABLE "project_favorite" DROP CONSTRAINT project_favorite_pkey;
ALTER TABLE "project_favorite" ADD PRIMARY KEY (project_id, user_id);
ALTER TABLE "project_favorite" DROP COLUMN authentified_user_id;

ALTER TABLE "workflow_favorite" DROP CONSTRAINT workflow_favorite_pkey;
ALTER TABLE "workflow_favorite" ADD PRIMARY KEY (workflow_id, user_id);
ALTER TABLE "workflow_favorite" DROP COLUMN authentified_user_id;

ALTER TABLE "user_timeline" DROP CONSTRAINT user_timeline_pkey;
ALTER TABLE "user_timeline" ADD PRIMARY KEY (user_id);
ALTER TABLE "user_timeline" DROP COLUMN authentified_user_id;

ALTER TABLE "workflow_template_bulk" DROP COLUMN authentified_user_id;
