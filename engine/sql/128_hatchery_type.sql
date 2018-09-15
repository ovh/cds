-- +migrate Up

ALTER TABLE hatchery 
    ADD COLUMN worker_model_id BIGINT,
    ADD COLUMN model_type VARCHAR(50) DEFAULT '',
    ADD COLUMN type VARCHAR(20) DEFAULT '',
    ADD COLUMN ratio_service INT DEFAULT 0,
    DROP COLUMN status,
    DROP COLUMN last_beat;

DROP TABLE hatchery_model;

SELECT create_foreign_key('FK_WORKER_MODEL_ID_HATCHERY', 'hatchery', 'worker_model', 'worker_model_id', 'id');

-- +migrate Down

ALTER TABLE hatchery 
    DROP COLUMN worker_model_id,
    DROP COLUMN model_type,
    DROP COLUMN type,
    DROP COLUMN ratio_service,
    ADD COLUMN status TEXT,
    ADD COLUMN last_beat TIMESTAMP WITH TIME ZONE;

CREATE TABLE IF NOT EXISTS "hatchery_model" (hatchery_id BIGINT, worker_model_id BIGINT, PRIMARY KEY(hatchery_id, worker_model_id));
select create_foreign_key('FK_HATCHERY_MODEL_HATCHERY_ID', 'hatchery_model', 'hatchery', 'hatchery_id', 'id');
select create_foreign_key('FK_HATCHERY_MODEL_WORKER_MODEL_ID', 'hatchery_model', 'worker_model', 'worker_model_id', 'id');
