-- +migrate Up

CREATE TABLE environment_key_tmp AS SELECT * FROM environment_key;
 ALTER TABLE environment_key_tmp ADD PRIMARY KEY (id);

ALTER TABLE "environment_key" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "environment_key" ADD COLUMN IF NOT EXISTS signer TEXT;

-- +migrate Down

ALTER TABLE "environment_key" DROP COLUMN sig;
ALTER TABLE "environment_key" DROP COLUMN signer;

UPDATE environment_key 
SET environment_id      = environment_key_tmp.environment_id,
    name                = environment_key_tmp.name,
    type                = environment_key_tmp.type,
    public              = environment_key_tmp.public,
    key_id              = environment_key_tmp.key_id,
    private             =  environment_key_tmp.private
FROM environment_key_tmp
WHERE environment_key_tmp.id = environment_key.id;

DROP TABLE environment_key_tmp;
