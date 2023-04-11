-- +migrate Up
ALTER TABLE grpc_plugin ADD COLUMN inputs JSONB;

-- +migrate Down
ALTER TABLE grpc_plugin DROP COLUMN inputs;

