-- +migrate Up
DROP table "stats";
DROP table "activity";

-- +migrate Down
CREATE TABLE IF NOT EXISTS "stats" (day DATE PRIMARY KEY, build BIGINT, unit_test BIGINT, testing BIGINT, deployment BIGINT, max_building_worker BIGINT, max_building_pipeline BIGINT);
CREATE TABLE IF NOT EXISTS "activity" (day DATE, project_id BIGINT, application_id BIGINT, build BIGINT, unit_test BIGINT, testing BIGINT, deployment BIGINT, PRIMARY KEY(day, project_id, application_id));
