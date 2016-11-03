-- +migrate Up
CREATE TABLE template (id BIGSERIAL PRIMARY KEY, name TEXT, author TEXT, description TEXT, identifier TEXT, size INTEGER, perm INTEGER, md5sum TEXT, object_path TEXT);

SELECT create_unique_index('template', 'IDX_TEMPLATE_IDENTIFIER', 'identifier');

SELECT create_unique_index('template', 'IDX_TEMPLATE_NAME', 'name');

GRANT SELECT, INSERT, UPDATE, DELETE on ALL TABLES IN SCHEMA public TO "cds";

GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO "cds";

-- +migrate Down
DROP TABLE template;
