-- +migrate Up

ALTER TABLE services DROP constraint services_pkey;
ALTER TABLE services ADD COLUMN id BIGSERIAL PRIMARY KEY;

ALTER TABLE hatchery 
    ADD COLUMN model_type VARCHAR(50) DEFAULT '',
    ADD COLUMN type VARCHAR(20) DEFAULT '',
    ADD COLUMN ratio_service INT DEFAULT 0,
    ADD COLUMN service_id BIGINT,
    DROP COLUMN status,
    DROP COLUMN last_beat;

SELECT create_foreign_key_idx_cascade('FK_HATCHERY_SERVICE', 'hatchery', 'services', 'service_id', 'id');

DROP TABLE hatchery_model;

ALTER TABLE workflow_node_run_job ADD COLUMN contains_service BOOLEAN DEFAULT false;

-- +migrate Down

ALTER TABLE hatchery 
    DROP COLUMN model_type,
    DROP COLUMN type,
    DROP COLUMN ratio_service,
    DROP COLUMN service_id,
    ADD COLUMN status TEXT,
    ADD COLUMN last_beat TIMESTAMP WITH TIME ZONE;

CREATE TABLE IF NOT EXISTS "hatchery_model" (hatchery_id BIGINT, worker_model_id BIGINT, PRIMARY KEY(hatchery_id, worker_model_id));
select create_foreign_key('FK_HATCHERY_MODEL_HATCHERY_ID', 'hatchery_model', 'hatchery', 'hatchery_id', 'id');
select create_foreign_key('FK_HATCHERY_MODEL_WORKER_MODEL_ID', 'hatchery_model', 'worker_model', 'worker_model_id', 'id');

ALTER TABLE workflow_node_run_job DROP COLUMN contains_service;

ALTER TABLE services DROP COLUMN id;
select create_primary_key('services', 'name');