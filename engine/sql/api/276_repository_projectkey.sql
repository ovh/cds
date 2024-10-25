-- +migrate Up
ALTER TABLE project_repository ADD COLUMN project_key TEXT;
UPDATE project_repository SET project_key = project.projectkey
  FROM vcs_project
  JOIN project ON project.id = vcs_project.project_id
  WHERE vcs_project.id = project_repository.vcs_project_id;

SELECT create_foreign_key_idx_cascade('fk_project_repository_project', 'project_repository', 'project', 'project_key', 'projectkey');

-- +migrate Down
ALTER TABLE project_repository DROP COLUMN project_key;

