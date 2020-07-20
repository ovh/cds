-- +migrate Up
CREATE TABLE IF NOT EXISTS workflow_as_code_events
(
    id BIGSERIAL PRIMARY KEY,
    workflow_id BIGINT,
    pullrequest_id BIGINT,
    pullrequest_url VARCHAR(500),
    username VARCHAR(255),
    creation_date TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP
);
SELECT create_foreign_key_idx_cascade('FK_AS_CODE_EVENT', 'workflow_as_code_events', 'workflow', 'workflow_id', 'id');

-- +migrate Down
DROP TABLE IF EXISTS workflow_as_code_events;
