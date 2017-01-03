-- +migrate Up

CREATE TABLE pipeline_scheduler (
        id BIGSERIAL PRIMARY KEY,
        application_id BIGINT NOT NULL,
        pipeline_id BIGINT NOT NULL,
        environment_id BIGINT NOT NULL,
        args JSONB,
        crontab TEXT NOT NULL,
        disable BOOLEAN DEFAULT FALSE
);

CREATE TABLE pipeline_scheduler_execution (
    id BIGSERIAL PRIMARY KEY,
    pipeline_scheduler_id BIGINT NOT NULL, 
    execution_planned_date TIMESTAMP WITH TIME ZONE,
    execution_date TIMESTAMP WITH TIME ZONE,
    executed BOOLEAN NOT NULL DEFAULT FALSE,
    pipeline_build_version BIGINT
);

SELECT create_foreign_key('FK_PIPELINE_SCHEDULER_APPLICATION', 'pipeline_scheduler', 'application', 'application_id', 'id');
SELECT create_foreign_key('FK_PIPELINE_SCHEDULER_PIPELINE', 'pipeline_scheduler', 'pipeline', 'pipeline_id', 'id');
SELECT create_foreign_key('FK_PIPELINE_SCHEDULER_ENVIRONMENT', 'pipeline_scheduler', 'environment', 'environment_id', 'id');
SELECT create_foreign_key('FK_PIPELINE_SCHEDULER_EXECUTION_PIPELINE_SCHEDULER', 'pipeline_scheduler_execution', 'pipeline_scheduler', 'pipeline_scheduler_id', 'id');

-- +migrate Down

DROP TABLE pipeline_scheduler_execution;
DROP TABLE pipeline_scheduler;
