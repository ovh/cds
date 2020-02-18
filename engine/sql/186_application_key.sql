-- +migrate Up

CREATE TABLE application_key_tmp AS SELECT * FROM application_key;
 ALTER TABLE application_key_tmp ADD PRIMARY KEY (id);

ALTER TABLE "application_key" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "application_key" ADD COLUMN IF NOT EXISTS signer TEXT;

-- +migrate Down

ALTER TABLE "application_key" DROP COLUMN sig;
ALTER TABLE "application_key" DROP COLUMN signer;

UPDATE application_key 
SET application_id      = application_key_tmp.application_id,
    name                = application_key_tmp.name,
    type                = application_key_tmp.type,
    public              = application_key_tmp.public,
    key_id              = application_key_tmp.key_id,
    private             =  application_key_tmp.private
FROM application_key_tmp
WHERE application_key_tmp.id = application_key.id;

DROP TABLE application_key_tmp;
