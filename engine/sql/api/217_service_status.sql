-- +migrate Up

alter table "service" drop column monitoring_status;

-- +migrate Down

alter table "service" add column monitoring_status JSONB;
