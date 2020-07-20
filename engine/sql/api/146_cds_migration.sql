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
    done TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP,
    major BIGINT,
    minor BIGINT,
    patch BIGINT
);
SELECT create_unique_index('cds_migration', 'IDX_CDS_MIGRATION_NAME_RELEASE_UNIQ', 'name,release');
select create_index('cds_migration','IDX_CDS_MIGRATION_STATUS', 'status');

CREATE TABLE cds_version
(
    id BIGSERIAL PRIMARY KEY,
    release VARCHAR(255) NOT NULL,
    major BIGINT,
    minor BIGINT,
    patch BIGINT,
    created TIMESTAMP WITH TIME ZONE DEFAULT LOCALTIMESTAMP
);
SELECT create_unique_index('cds_version', 'IDX_CDS_VERSION_RELEASE_UNIQ', 'release');

-- +migrate Down
DROP TABLE cds_migration;
DROP TABLE cds_version;
