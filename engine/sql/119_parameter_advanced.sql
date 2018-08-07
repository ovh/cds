-- +migrate Up
ALTER TABLE action_edge_parameter ADD COLUMN advanced BOOLEAN;
ALTER TABLE action_parameter ADD COLUMN advanced BOOLEAN;
update action_parameter set advanced = false;
update action_edge_parameter set advanced = false;

-- +migrate Down

ALTER TABLE action_edge_parameter DROP COLUMN advanced;
ALTER TABLE action_parameter DROP COLUMN advanced;
