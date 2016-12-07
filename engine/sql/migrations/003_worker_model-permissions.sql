-- +migrate Up

ALTER TABLE worker_model DROP COLUMN group_id;
ALTER TABLE worker_model ADD COLUMN group_id BIGINT;
ALTER TABLE worker_model ADD COLUMN created_by JSONB;

UPDATE worker_model set group_id = (SELECT id FROM "group" WHERE name = 'shared.infra' LIMIT 1);
ALTER TABLE worker_model ALTER group_id TYPE BIGINT; 
ALTER TABLE worker_model ALTER group_id SET NOT NULL;

select create_index('worker_model','IDX_WORKER_MODEL_GROUP_ID','group_id');
SELECT create_foreign_key('FK_WORKER_MODEL_GROUP', 'worker_model', 'group', 'group_id', 'id');

-- +migrate Down

ALTER TABLE worker_model DROP COLUMN group_id;
ALTER TABLE worker_model DROP COLUMN created_by;
