-- +migrate Up
ALTER TABLE "poller_execution" RENAME TO "poller_execution_old";
ALTER TABLE "poller_execution_old" DROP CONSTRAINT IF EXISTS "fk_poller_execution_application";
ALTER TABLE "poller_execution_old" DROP CONSTRAINT IF EXISTS "fk_poller_execution_pipeline";

CREATE TABLE "poller_execution" (id BIGSERIAL PRIMARY KEY, application_id BIGINT NOT NULL, pipeline_id BIGINT NOT NULL, execution_planned_date TIMESTAMP WITH TIME ZONE, execution_date TIMESTAMP WITH TIME ZONE, executed BOOLEAN NOT NULL DEFAULT FALSE, push_events JSONB, pipeline_build_versions JSONB, error TEXT);
select create_foreign_key('FK_POLLER_EXECUTION_APPLICATION', 'poller_execution', 'application', 'application_id', 'id');
select create_foreign_key('FK_POLLER_EXECUTION_PIPELINE', 'poller_execution', 'pipeline', 'pipeline_id', 'id');

-- +migrate Down
DROP TABLE poller_execution;
ALTER TABLE "poller_execution_old" RENAME TO "poller_execution";
select create_foreign_key('FK_POLLER_EXECUTION_APPLICATION', 'poller_execution', 'application', 'application_id', 'id');
select create_foreign_key('FK_POLLER_EXECUTION_PIPELINE', 'poller_execution', 'pipeline', 'pipeline_id', 'id');
