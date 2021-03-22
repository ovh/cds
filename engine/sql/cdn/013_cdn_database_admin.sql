-- +migrate Up
CREATE TABLE "test_encrypted_data" (
  id BIGSERIAL PRIMARY KEY,
  data TEXT,
  sensitive_data BYTEA,
  another_sensitive_data BYTEA,
  sig BYTEA,
  signer TEXT
);

-- +migrate Down
DROP TABLE "test_encrypted_data";