-- +migrate Up notransaction
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_item_worker_cache_expire
ON item (((api_ref->>'expire_at')::timestamptz) ASC, created ASC)
WHERE type IN ('worker-cache', 'worker-cache-v2')
  AND to_delete = false;

-- +migrate Down
DROP INDEX IF EXISTS idx_item_worker_cache_expire;
