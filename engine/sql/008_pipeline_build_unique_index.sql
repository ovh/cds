-- +migrate Up
drop index IDX_PIPELINE_BUILD_UNIQUE_BUILD_NUMBER;
select create_unique_index('pipeline_build', 'IDX_PIPELINE_BUILD_UNIQUE_BUILD_NUMBER', 'build_number,pipeline_id,application_id,environment_id');

-- +migrate Down
drop index IDX_PIPELINE_BUILD_UNIQUE_BUILD_NUMBER;
select create_index('pipeline_build', 'IDX_PIPELINE_BUILD_UNIQUE_BUILD_NUMBER', 'build_number,pipeline_id,application_id,environment_id');