-- +migrate Up
ALTER TABLE grpc_plugin ADD COLUMN inputs JSONB DEFAULT '{}';

-- +migrate Down
ALTER TABLE grpc_plugin DROP COLUMN inputs;

