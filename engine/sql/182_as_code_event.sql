-- +migrate Up
CREATE TABLE IF NOT EXISTS as_code_events
(
    id BIGSERIAL PRIMARY KEY,
    pullrequest_id BIGINT,
    pullrequest_url VARCHAR(500),
    username VARCHAR(255),
    creation_date TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    from_repository VARCHAR(300),
    migrate BOOL DEFAULT false,
    data JSONB
);
CREATE INDEX idx_as_code_events_data ON as_code_events USING gin (data);
ALTER TABLE workflow_as_code_events DROP CONSTRAINT FK_AS_CODE_EVENT;

-- +migrate Down
DROP TABLE IF EXISTS as_code_events;
SELECT create_foreign_key_idx_cascade('FK_AS_CODE_EVENT', 'workflow_as_code_events', 'workflow', 'workflow_id', 'id');

