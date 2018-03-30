-- +migrate Up
CREATE TABLE IF NOT EXISTS project_favorite (user_id BIGINT, project_id BIGINT, PRIMARY KEY(user_id, project_id));
SELECT create_foreign_key('FK_PROJECT_FAVORITE_APPLICATION', 'project_favorite', 'user', 'user_id', 'id');
SELECT create_foreign_key('FK_PROJECT_FAVORITE_GROUP', 'project_favorite', 'project', 'project_id', 'id');

CREATE TABLE IF NOT EXISTS workflow_favorite (user_id BIGINT, workflow_id BIGINT, PRIMARY KEY(user_id, workflow_id));
SELECT create_foreign_key('FK_WORKFLOW_FAVORITE_APPLICATION', 'workflow_favorite', 'user', 'user_id', 'id');
SELECT create_foreign_key('FK_WORKFLOW_FAVORITE_GROUP', 'workflow_favorite', 'workflow', 'workflow_id', 'id');

-- +migrate Down
DROP TABLE IF EXISTS project_favorite, workflow_favorite;
