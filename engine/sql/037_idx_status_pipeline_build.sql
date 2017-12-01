-- +migrate Up
select create_index('pipeline_build', 'IDX_PIPELINE_BUILD_STATUS', 'status');

-- +migrate Down
drop index IDX_PIPELINE_BUILD_STATUS;
