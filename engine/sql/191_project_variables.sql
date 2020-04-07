-- +migrate Up

CREATE TABLE project_variable_tmp AS SELECT * FROM project_variable;
ALTER TABLE project_variable_tmp ADD PRIMARY KEY (id);

ALTER TABLE "project_variable" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "project_variable" ADD COLUMN IF NOT EXISTS signer TEXT;

-- +migrate Down

ALTER TABLE "project_variable" DROP COLUMN sig;
ALTER TABLE "project_variable" DROP COLUMN signer;

UPDATE project_variable 
SET application_id      = project_variable_tmp.application_id,
    var_name            = project_variable_tmp.var_name,
    var_value           = project_variable_tmp.var_value,
    cipher_value        = project_variable_tmp.cipher_value,
    var_type            = project_variable_tmp.var_type,
FROM project_variable_tmp
WHERE project_variable_tmp.id = project_variable.id;

DROP TABLE project_variable_tmp;