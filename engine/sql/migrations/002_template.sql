-- +migrate Up
CREATE TABLE template (id BIGSERIAL PRIMARY KEY, name TEXT, type TEXT,author TEXT, description TEXT, identifier TEXT, size INTEGER, perm INTEGER, md5sum TEXT, object_path TEXT);

CREATE TABLE template_params (template_id BIGINT, params JSONB);

SELECT create_unique_index('template', 'IDX_TEMPLATE_IDENTIFIER', 'identifier');

SELECT create_unique_index('template', 'IDX_TEMPLATE_NAME', 'name');

select create_foreign_key('FK_TEMPLATE_PARAMS_TEMPLATE', 'template_params', 'template', 'template_id', 'id');

GRANT SELECT, INSERT, UPDATE, DELETE on ALL TABLES IN SCHEMA public TO "cds";

GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO "cds";

-- +migrate Down
DROP TABLE template_params;
DROP TABLE template;
