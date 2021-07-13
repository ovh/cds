-- +migrate Up
DROP INDEX "idx_log_item";
DROP INDEX "idx_log_project_key";
DROP INDEX "idx_api_ref";
DROP INDEX "idx_storage_unit_item_hash_locator";
DROP INDEX "idx_item_status_type";

-- +migrate Down
CREATE INDEX idx_log_item ON item(type, (api_ref->>'job_id'), (api_ref->>'step_order'));
CREATE INDEX idx_log_project_key ON item((api_ref->>'project_key'));
CREATE INDEX idx_api_ref ON "item" USING GIN (api_ref);
SELECT create_index('storage_unit_item', 'idx_storage_unit_id_hash_locator', 'unit_id,hash_locator');
SELECT create_index('item', 'idx_item_status_type', 'status,type');
