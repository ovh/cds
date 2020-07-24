-- +migrate Up
CREATE TABLE IF NOT EXISTS "feature_flipping" (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    rule TEXT NOT NULL,
    created TIMESTAMP WITH TIME ZONE DEFAULT current_timestamp,
    updated TIMESTAMP WITH TIME ZONE DEFAULT current_timestamp
);

select create_unique_index('feature_flipping', 'IDX_FEATURE_FLIPPING', 'name');

-- +migrate Down
DROP TABLE "feature_flipping";

