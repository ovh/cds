-- Your SQL goes here
CREATE TABLE IF NOT EXISTS "run" (
    id BIGSERIAL PRIMARY KEY,
    run_id BIGINT NOT NULL,
    num BIGINT NOT NULL,
    project_key TEXT NOT NULL,
    workflow_name TEXT NOT NULL,
    branch TEXT DEFAULT '',
    status TEXT NOT NULL,
    updated TIMESTAMP WITH TIME ZONE DEFAULT current_timestamp
);