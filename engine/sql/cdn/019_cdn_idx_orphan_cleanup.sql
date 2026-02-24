-- +migrate Up notransaction
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_item_v1_orphan_cleanup
ON item (created ASC)
WHERE type IN ('step-log', 'service-log', 'run-result')
  AND to_delete = false;

-- +migrate Down
DROP INDEX IF EXISTS idx_item_v1_orphan_cleanup;
