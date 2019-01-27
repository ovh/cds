-- +migrate Up

ALTER TABLE grpc_plugin ADD COLUMN platform_model_id BIGINT;
UPDATE grpc_plugin set platform_model_id=(select id from platform_model where grpc_plugin_id=grpc_plugin.id); 
ALTER TABLE platform_model DROP COLUMN grpc_plugin_id;
SELECT create_foreign_key_idx_cascade('FK_GRPC_PLUGIN_PLATFORM_MODEL', 'grpc_plugin', 'platform_model', 'platform_model_id', 'id');
UPDATE grpc_plugin set type = 'platform-deploy_application' where type = 'platform';

-- +migrate Down
ALTER TABLE platform_model ADD COLUMN grpc_plugin_id BIGINT;
UPDATE platform_model set grpc_plugin_id=(select id from grpc_plugin where platform_model_id=platform_model.id);
SELECT create_foreign_key_idx_cascade('FK_PLATFORM_MODEL_GRPC_PLUGIN', 'platform_model', 'grpc_plugin', 'grpc_plugin_id', 'id');
ALTER TABLE grpc_plugin DROP COLUMN platform_model_id;
UPDATE grpc_plugin set type = 'platform' where type = 'platform-deploy_application';
