-- +migrate Up
ALTER TABLE "pipeline_scheduler_execution"  DROP CONSTRAINT fk_pipeline_scheduler_execution_pipeline_scheduler;

-- +migrate StatementBegin
ALTER TABLE "pipeline_scheduler_execution" 
    ADD CONSTRAINT fk_pipeline_scheduler_execution_pipeline_scheduler 
    FOREIGN KEY (pipeline_scheduler_id) REFERENCES pipeline_scheduler(id) ON DELETE CASCADE;
-- +migrate StatementEnd

-- +migrate Down
ALTER TABLE "pipeline_scheduler_execution"  DROP CONSTRAINT fk_pipeline_scheduler_execution_pipeline_scheduler;
SELECT create_foreign_key('FK_PIPELINE_SCHEDULER_EXECUTION_PIPELINE_SCHEDULER', 'pipeline_scheduler_execution', 'pipeline_scheduler', 'pipeline_scheduler_id', 'id');
