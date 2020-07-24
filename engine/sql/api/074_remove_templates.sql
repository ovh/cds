-- +migrate Up
DROP TABLE template_params;
DROP TABLE template;

-- +migrate Down
SELECT 1;
