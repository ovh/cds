-- +migrate Up

CREATE TABLE environment_variable_tmp AS SELECT * FROM environment_variable;
ALTER TABLE environment_variable_tmp ADD PRIMARY KEY (id);

ALTER TABLE "environment_variable" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "environment_variable" ADD COLUMN IF NOT EXISTS signer TEXT;

-- +migrate Down

ALTER TABLE "environment_variable" DROP COLUMN sig;
ALTER TABLE "environment_variable" DROP COLUMN signer;

UPDATE environment_variable 
SET application_id      = environment_variable_tmp.application_id,
    "name"            = environment_variable_tmp.name,
    "value"           = environment_variable_tmp.value,
    cipher_value        = environment_variable_tmp.cipher_value,
    "type"            = environment_variable_tmp.type,
FROM environment_variable_tmp
WHERE environment_variable_tmp.id = environment_variable.id;

DROP TABLE environment_variable_tmp;