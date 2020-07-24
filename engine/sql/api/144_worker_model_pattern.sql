-- +migrate Up

update worker_model set model=replace(model::TEXT, '--booked-pb-job-id={{.PipelineBuildJobID}}', '')::jsonb;
update worker_model set model=replace(model::TEXT, 'export CDS_BOOKED_PB_JOB_ID={{.PipelineBuildJobID}}\n', '')::jsonb;
update worker_model_pattern set model=replace(model::TEXT, '--booked-pb-job-id={{.PipelineBuildJobID}}', '')::jsonb;
update worker_model_pattern set model=replace(model::TEXT, 'export CDS_BOOKED_PB_JOB_ID={{.PipelineBuildJobID}}\n', '')::jsonb;

-- +migrate Down

select 1;
