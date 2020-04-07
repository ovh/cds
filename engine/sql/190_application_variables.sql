-- +migrate Up

CREATE TABLE application_variable_tmp AS SELECT * FROM application_variable;
ALTER TABLE application_variable_tmp ADD PRIMARY KEY (id);

ALTER TABLE "application_variable" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "application_variable" ADD COLUMN IF NOT EXISTS signer TEXT;

-- +migrate Down

ALTER TABLE "application_variable" DROP COLUMN sig;
ALTER TABLE "application_variable" DROP COLUMN signer;

UPDATE application_variable 
SET application_id      = application_variable_tmp.application_id,
    var_name            = application_variable_tmp.var_name,
    var_value           = application_variable_tmp.var_value,
    cipher_value        = application_variable_tmp.cipher_value,
    var_type            = application_variable_tmp.var_type,
FROM application_variable_tmp
WHERE application_variable_tmp.id = application_variable.id;

DROP TABLE application_variable_tmp;