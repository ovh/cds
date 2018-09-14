-- +migrate Up

ALTER TABLE hatchery 
    ADD COLUMN model_type VARCHAR(50) DEFAULT '',
    ADD COLUMN type VARCHAR(20) DEFAULT '',
    ADD COLUMN ratio_service INT DEFAULT 0;

-- +migrate Down

ALTER TABLE hatchery 
    DROP COLUMN model_type,
    DROP COLUMN type,
    DROP COLUMN ratio_service;
