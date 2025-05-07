-- +migrate Up
CREATE TABLE IF NOT EXISTS entity_favorite (
    "authentified_user_id" VARCHAR(36),
    "project_repository_id" uuid NOT NULL,
    "type" VARCHAR(255),
    "name" VARCHAR(255),
    PRIMARY KEY("authentified_user_id", "type", "name")
);
SELECT create_foreign_key_idx_cascade('FK_ENTITY_FAVORITE_AUTHENTIFIED_USER', 'entity_favorite', 'authentified_user', 'authentified_user_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ENTITY_FAVORITE_REPOSITORY', 'entity_favorite', 'project_repository', 'project_repository_id', 'id');

-- +migrate Down
DROP TABLE IF EXISTS entity_favorite;