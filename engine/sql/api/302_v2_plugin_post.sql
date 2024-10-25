-- +migrate Up
ALTER TABLE grpc_plugin ADD COLUMN post JSONB DEFAULT '{}';

-- +migrate Down
ALTER TABLE grpc_plugin DROP COLUMN post;
