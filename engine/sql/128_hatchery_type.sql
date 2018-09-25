-- +migrate Up

DROP TABLE services;
DROP TABLE hatchery_model;
DROP TABLE hatchery;

ALTER TABLE workflow_node_run_job
    ADD COLUMN model_type TEXT,
    ADD COLUMN contains_service BOOLEAN DEFAULT false;

UPDATE workflow_node_run_job set contains_service = false;

CREATE TABLE services (
    id BIGSERIAL PRIMARY KEY,
    name character varying(256) NOT NULL,
    type character varying(256) NOT NULL,
    http_url character varying(256) NOT NULL,
    last_heartbeat timestamp with time zone NOT NULL,
    hash character varying(256) NOT NULL,
    monitoring_status jsonb,
    group_id bigint
);

-- +migrate Down

CREATE TABLE IF NOT EXISTS hatchery (
    id bigint NOT NULL primary key,
    name text,
    uid text,
    group_id integer,
    status text,
    last_beat timestamp with time zone
);

SELECT create_unique_index('hatchery', 'IDX_HATCHERY_NAME', 'name');

CREATE TABLE IF NOT EXISTS "hatchery_model" (hatchery_id BIGINT, worker_model_id BIGINT, PRIMARY KEY(hatchery_id, worker_model_id));
select create_foreign_key('FK_HATCHERY_MODEL_HATCHERY_ID', 'hatchery_model', 'hatchery', 'hatchery_id', 'id');
select create_foreign_key('FK_HATCHERY_MODEL_WORKER_MODEL_ID', 'hatchery_model', 'worker_model', 'worker_model_id', 'id');

ALTER TABLE workflow_node_run_job
    DROP COLUMN model_type,
    DROP COLUMN contains_service;

DROP TABLE services;

CREATE TABLE services (
    name character varying(256) NOT NULL PRIMARY KEY,
    type character varying(256) NOT NULL,
    http_url character varying(256) NOT NULL,
    last_heartbeat timestamp with time zone NOT NULL,
    hash character varying(256) NOT NULL,
    monitoring_status jsonb
);
