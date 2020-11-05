-- +migrate Up
alter table "workflow" drop column purge_tags;
alter table "workflow" drop column history_length;


-- +migrate Down
ALTER TABLE workflow ADD COLUMN purge_tags JSONB;
ALTER TABLE workflow ADD COLUMN history_length BIGINT DEFAULT 20;
