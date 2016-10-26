-- +migrate Up
CREATE TABLE IF NOT EXISTS gorp_migrations_lock (id TEXT, locked TIMESTAMP WITH TIME ZONE, unlocked TIMESTAMP WITH TIME ZONE);

-- +migrate Down
DROP TABLE gorp_migrations_lock;