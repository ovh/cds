-- +migrate Up
CREATE TABLE IF NOT EXISTS "services" (
    name VARCHAR(256) NOT NULL,
    type VARCHAR(256) NOT NULL,
    http_url VARCHAR(256) NOT NULL,
    last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL,
    hash VARCHAR(256) NOT NULL,
    PRIMARY KEY (name)
);
-- +migrate Down
DROP TABLE services;
