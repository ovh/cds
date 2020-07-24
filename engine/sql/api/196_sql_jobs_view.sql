-- +migrate Up
CREATE VIEW view_node_run_job AS 
SELECT  
        jobs->>'id' as "id",
        jobs->>'workflow_node_run_id' as "workflow_node_run_id",
        jobs->>'status' as "status",
        cast(jobs->>'start' as timestamp with time zone) as "started", 
        cast(jobs->>'queued' as timestamp with time zone) as "queued" ,
        cast(jobs->>'done' as timestamp with time zone) as "done",  
        jobs->>'retry' as "retry",
        jobs->>'model' as "model",
        jobs->>'worker_name' as "worker",
        jobs->>'hatchery_name' as "hatchery"
FROM (
    SELECT 
        jsonb_array_elements(
            CASE jsonb_typeof(stages->'run_jobs') 
            WHEN 'array' THEN stages->'run_jobs'
            ELSE '[]' END
        ) jobs
    FROM (
        SELECT 
        jsonb_array_elements(
            CASE jsonb_typeof(stages) 
            WHEN 'array' THEN stages 
            ELSE '[]' END
        ) stages
        FROM workflow_node_run
    ) tmpStages
) tmpJobs
order by started desc;

-- +migrate Down
DROP VIEW view_node_run_job;
