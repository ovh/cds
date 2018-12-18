-- +migrate Up
CREATE TABLE cds_migration
(
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(255) NOT NULL,
    release VARCHAR(255) NOT NULL,
    progress TEXT,
    error TEXT,
    mandatory BOOLEAN DEFAULT false,
    created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    done TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP
);

SELECT create_unique_index('cds_migration', 'IDX_CDS_MIGRATION_NAME_UNIQ', 'name');
-- +migrate Down
DROP TABLE cds_migration;
