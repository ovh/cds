-- +migrate Up
UPDATE action_edge SET exec_order = exec_order + 100 WHERE final = true;
ALTER TABLE action_edge ADD COLUMN optional BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE action_edge ADD COLUMN always_executed BOOLEAN NOT NULL DEFAULT false;
UPDATE action_edge SET always_executed = true WHERE final = true;
ALTER TABLE action_edge DROP COLUMN final;

-- +migrate Down
ALTER TABLE action_edge ADD COLUMN final BOOLEAN NOT NULL DEFAULT false;
UPDATE action_edge SET final = true, exec_order = exec_order - 100 WHERE exec_order > 100 AND always_executed = true;
ALTER TABLE action_edge DROP COLUMN optional;
ALTER TABLE action_edge DROP COLUMN always_executed;
