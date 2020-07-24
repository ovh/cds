-- +migrate Up
update pipeline_parameter set name = 'param_' || id where name = '';

-- +migrate Down
select 1;
