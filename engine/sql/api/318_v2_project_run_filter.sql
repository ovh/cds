-- +migrate Up
CREATE TABLE project_run_filter (
    "id"            uuid PRIMARY KEY,
    "project_key"   VARCHAR(255) NOT NULL,
    "name"          VARCHAR(100) NOT NULL,
    "value"         TEXT NOT NULL,
    "sort"          VARCHAR(50) NOT NULL DEFAULT '',
    "order"         BIGINT NOT NULL DEFAULT 0,
    "last_modified" TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP
);

-- Création de la foreign key vers project avec cascade
SELECT create_foreign_key_idx_cascade(
    'FK_v2_project_run_filter_project',
    'project_run_filter',
    'project',
    'project_key',
    'projectkey'
);

-- Création de l'index unique sur (project_key, name)
SELECT create_unique_index(
    'project_run_filter',
    'idx_unq_project_run_filter',
    'project_key,name'
);

-- +migrate Down
DROP TABLE project_run_filter;
