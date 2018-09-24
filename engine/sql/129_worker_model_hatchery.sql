-- +migrate Up

update worker_model set model=replace(model::TEXT, '--hatchery={{.Hatchery}}', '')::jsonb;
update worker_model_pattern set model=replace(model::TEXT, '--hatchery={{.Hatchery}}', '')::jsonb;
update worker_model_pattern set model=replace(model::TEXT, 'export CDS_HATCHERY={{.Hatchery}}\n', '')::jsonb;

-- +migrate Down

select 1;
