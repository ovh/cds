-- +migrate Up

CREATE TABLE project_key_tmp AS SELECT * FROM project_key;
ALTER TABLE project_key_tmp ADD PRIMARY KEY (id);

ALTER TABLE "project_key" ADD COLUMN IF NOT EXISTS sig BYTEA;
ALTER TABLE "project_key" ADD COLUMN IF NOT EXISTS signer TEXT;

-- +migrate Down

ALTER TABLE "project_key" DROP COLUMN sig;
ALTER TABLE "project_key" DROP COLUMN signer;

UPDATE project_key 
SET project_id      = project_key_tmp.project_id,
    name                = project_key_tmp.name,
    type                = project_key_tmp.type,
    public              = project_key_tmp.public,
    key_id              = project_key_tmp.key_id,
    private             =  project_key_tmp.private
FROM project_key_tmp
WHERE project_key_tmp.id = project_key.id;

DROP TABLE project_key_tmp;
