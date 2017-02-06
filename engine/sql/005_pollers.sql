-- +migrate Up
ALTER TABLE "poller_execution" RENAME TO "poller_execution_old";
CREATE TABLE "poller_execution" (id BIGSERIAL PRIMARY KEY, application_id BIGINT NOT NULL, pipeline_id BIGINT NOT NULL, execution_planned_date TIMESTAMP WITH TIME ZONE, execution_date TIMESTAMP WITH TIME ZONE, executed BOOLEAN NOT NULL DEFAULT FALSE, push_events JSONB, pipeline_build_versions JSONB);


-- +migrate Down
DROP TABLE poller_execution;
ALTER TABLE "poller_execution_old" RENAME TO "poller_execution";
