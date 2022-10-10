-- +migrate Up
ALTER TABLE auth_consumer RENAME TO auth_consumer_old;
CREATE TABLE "auth_consumer" (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(255) NOT NULL,
    parent_id VARCHAR(36),
    type VARCHAR(64),
    created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    issued_at TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    disabled BOOLEAN NOT NULL DEFAULT FALSE,
    warnings JSONB,
    validity_periods JSONB,
    last_authentication TIMESTAMP WITH TIME ZONE,
    signer TEXT,
    sig BYTEA
);
SELECT create_foreign_key_idx_cascade('FK_AUTH_CONSUMER_PARENT_ID', 'auth_consumer', 'auth_consumer', 'parent_id', 'id');
CREATE TABLE "auth_consumer_user"
(
    id                uuid PRIMARY KEY,
    auth_consumer_id  VARCHAR(36),
    user_id           VARCHAR(36),
    data              JSONB,
    group_ids         JSONB,
    invalid_group_ids JSONB,
    scope_details     JSONB,
    service_name      VARCHAR (256),
    service_type      VARCHAR(256),
    service_region    VARCHAR(256),
    service_ignore_job_with_no_region BOOLEAN,
    signer TEXT,
    sig BYTEA
);
SELECT create_foreign_key_idx_cascade('FK_AUTH_CONSUMER_USER_ID', 'auth_consumer_user', 'authentified_user', 'user_id', 'id');
SELECT create_foreign_key_idx_cascade('FK_AUTH_CONSUMER_USER_CONSUMER', 'auth_consumer_user', 'auth_consumer', 'auth_consumer_id', 'id');


ALTER TABLE auth_session DROP CONSTRAINT fk_auth_session_consumer;
DELETE FROM auth_session;
SELECT create_foreign_key_idx_cascade('FK_AUTH_SESSION_CONSUMER', 'auth_session', 'auth_consumer', 'consumer_id', 'id');

-- +migrate Down
DELETE FROM auth_session;
DROP TABLE auth_consumer_user;
DROP TABLE auth_consumer CASCADE;
ALTER TABLE auth_consumer_old RENAME TO auth_consumer;
SELECT create_foreign_key_idx_cascade('FK_AUTH_SESSION_CONSUMER', 'auth_session', 'auth_consumer', 'consumer_id', 'id');
