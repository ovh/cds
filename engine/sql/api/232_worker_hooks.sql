-- +migrate Up
CREATE TABLE "worker_hook_project_integration" (
    "id" BIGSERIAL PRIMARY KEY,
    "project_integration_id" BIGINT NOT NULL,
    "configuration" JSONB NOT NULL,
    "disable" BOOLEAN NOT NULL DEFAULT FALSE
);

ALTER TABLE "worker_hook_project_integration" 
    ADD CONSTRAINT FK_WORKER_HOOK_PROJECT_INTEGRATION_PROJECT_INTEGRATION_ID 
    FOREIGN KEY (project_integration_id) REFERENCES project_integration(id) ON DELETE CASCADE;

CREATE INDEX idx_worker_hook_project_integration_enable ON worker_hook_project_integration ("disable", "project_integration_id");

-- +migrate Down
DROP TABLE "worker_hook_project_integration";
