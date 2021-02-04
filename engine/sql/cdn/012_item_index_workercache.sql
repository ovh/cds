-- +migrate Up

CREATE INDEX idx_item_worker_cache ON item ((api_ref->>'cache_tag'));

-- +migrate Down
DROP index idx_item_worker_cache;

