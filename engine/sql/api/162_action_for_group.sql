
-- +migrate Up

-- remove unused data
DELETE FROM action_requirement WHERE id IN (
  SELECT actr.id FROM action_requirement AS actr
  LEFT JOIN "action" AS act ON act.id = actr.action_id
  WHERE act.id IS NULL
);

-- set empty string instead of null values
UPDATE action_parameter SET "description" = '' WHERE "description" IS NULL;
ALTER TABLE action_parameter ALTER COLUMN "description" SET NOT NULL;

-- remove unused table and column
DROP TABLE IF EXISTS template_action;
ALTER TABLE action_parameter DROP COLUMN IF EXISTS worker_model_name;
ALTER TABLE "action" DROP COLUMN IF EXISTS "data";

-- set Default type for actions with empty type and set not null constraints
UPDATE "action" SET "type" = 'Default' WHERE "type" = '';
ALTER TABLE "action" ALTER COLUMN "name" SET NOT NULL;
ALTER TABLE "action" ALTER COLUMN "type" SET NOT NULL;
ALTER TABLE "action" ALTER COLUMN "description" SET NOT NULL;

-- replace existing foreign keys with cascade ones
ALTER TABLE action_parameter DROP CONSTRAINT IF EXISTS "fk_action_parameter_action";
ALTER TABLE action_requirement DROP CONSTRAINT IF EXISTS "fk_action_requirement_action";
ALTER TABLE action_edge DROP CONSTRAINT IF EXISTS "fk_action_edge_parent_action";
ALTER TABLE action_edge_parameter DROP CONSTRAINT IF EXISTS "fk_action_edge_parameter_action_edge";
SELECT create_foreign_key_idx_cascade('FK_ACTION_PARAMETER_ACTION', 'action_parameter', 'action', 'action_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ACTION_REQUIREMENT_ACTION', 'action_requirement', 'action', 'action_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ACTION_EDGE_PARENT_ACTION', 'action_edge', 'action', 'parent_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_ACTION_EDGE_PARAMETER_ACTION_EDGE', 'action_edge_parameter', 'action_edge', 'action_edge_id', 'id');

-- change type for column name and type and add indexes (useful to check if action exists)
ALTER TABLE "action" ALTER COLUMN "name" TYPE VARCHAR(100) USING "name"::VARCHAR(100);
CREATE INDEX idx_action_name ON "action" ("name");
ALTER TABLE "action" ALTER COLUMN "type" TYPE VARCHAR(100) USING "type"::VARCHAR(100);
CREATE INDEX idx_action_type ON "action" ("type");

-- add column group_id on action and set to 1 (shared.infra) for all not joined action
ALTER TABLE "action" ADD COLUMN group_id BIGINT;
SELECT create_foreign_key_idx_cascade('FK_ACTION_GROUP', 'action', 'group', 'group_id', 'id');
UPDATE "action" SET group_id = shared.id FROM (SELECT id FROM "group" WHERE name = 'shared.infra') AS shared WHERE "type" = 'Default';

-- migrate action audits
DELETE FROM action_audit WHERE LOWER(change) = 'action delete';

ALTER TABLE action_audit DROP CONSTRAINT "action_audit_pkey";
ALTER TABLE action_audit ADD COLUMN id SERIAL PRIMARY KEY;

CREATE INDEX idx_action_audit_action_id ON action_audit ("action_id");

ALTER TABLE action_audit ADD COLUMN event_type VARCHAR(100);
CREATE INDEX idx_action_audit_event_type ON action_audit ("event_type");
UPDATE action_audit SET event_type = 'ActionAdd' WHERE LOWER(change) = 'action create';
UPDATE action_audit SET event_type = 'ActionUpdate' WHERE LOWER(change) = 'action update';
ALTER TABLE action_audit ADD COLUMN created TIMESTAMP WITH TIME ZONE;
UPDATE action_audit SET created = versionned;
ALTER TABLE action_audit ADD COLUMN triggered_by VARCHAR(100);
UPDATE action_audit SET triggered_by = u.username FROM "user" as u WHERE action_audit.user_id = u.id;
ALTER TABLE action_audit ADD COLUMN data_before TEXT;
UPDATE action_audit SET data_before = action_json::TEXT;
ALTER TABLE action_audit ADD COLUMN data_after TEXT;
UPDATE action_audit SET data_after = action_json::TEXT;
ALTER TABLE action_audit ADD COLUMN data_type VARCHAR(20);
UPDATE action_audit SET data_type = 'json';

ALTER TABLE action_audit ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE action_audit ALTER COLUMN versionned DROP NOT NULL;

-- TODO remove action.public, action_audit.change, action_audit.user_id, action_audit.versionned, action_audit.action_json in future script

-- +migrate Down

-- restore foreign keys
ALTER TABLE action_parameter DROP CONSTRAINT "fk_action_parameter_action";
ALTER TABLE action_requirement DROP CONSTRAINT "fk_action_requirement_action";
ALTER TABLE action_edge DROP CONSTRAINT "fk_action_edge_parent_action";
ALTER TABLE action_edge_parameter DROP CONSTRAINT "fk_action_edge_parameter_action_edge";
select create_foreign_key('FK_ACTION_PARAMETER_ACTION', 'action_parameter', 'action', 'action_id', 'id');
select create_foreign_key('FK_ACTION_REQUIREMENT_ACTION', 'action_requirement', 'action', 'action_id', 'id');
select create_foreign_key('FK_ACTION_EDGE_PARENT_ACTION', 'action_edge', 'action', 'parent_id', 'id');
select create_foreign_key('FK_ACTION_EDGE_PARAMETER_ACTION_EDGE', 'action_edge_parameter', 'action_edge', 'action_edge_id', 'id');

-- restore type for column name and type and remove indexes
ALTER TABLE "action" ALTER COLUMN "name" TYPE TEXT USING "name"::TEXT;
DROP INDEX IF EXISTS idx_action_name;
ALTER TABLE "action" ALTER COLUMN "type" TYPE TEXT USING "type"::TEXT;
DROP INDEX IF EXISTS idx_action_type;

-- remove group_id column
ALTER TABLE "action" DROP COLUMN group_id;

-- restore action audits
UPDATE action_audit SET user_id = u.id FROM "user" as u WHERE action_audit.triggered_by = u.username;
UPDATE action_audit SET versionned = created;
UPDATE action_audit SET action_json = data_after::JSONB;
UPDATE action_audit SET change = 'Action create' WHERE event_type = 'ActionAdd';
UPDATE action_audit SET change = 'Action update' WHERE event_type = 'ActionUpdate';

ALTER TABLE action_audit DROP COLUMN id;
select create_primary_key('action_audit', 'action_id,user_id,versionned');

DROP INDEX IF EXISTS idx_action_audit_action_id;

ALTER TABLE action_audit DROP COLUMN IF EXISTS event_type;
ALTER TABLE action_audit DROP COLUMN IF EXISTS created;
ALTER TABLE action_audit DROP COLUMN IF EXISTS triggered_by;
ALTER TABLE action_audit DROP COLUMN IF EXISTS data_before;
ALTER TABLE action_audit DROP COLUMN IF EXISTS data_after;
ALTER TABLE action_audit DROP COLUMN IF EXISTS data_type;

ALTER TABLE action_audit ALTER COLUMN user_id SET NOT NULL;
ALTER TABLE action_audit ALTER COLUMN versionned SET NOT NULL;
