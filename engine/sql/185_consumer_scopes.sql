-- +migrate Up
ALTER TABLE "auth_consumer" ADD COLUMN scope_details JSONB;

UPDATE auth_consumer SET scope_details = scopes WHERE scopes::TEXT <> 'null';

UPDATE auth_consumer
SET scope_details = tmp1.scope_details
FROM (
	SELECT
    tmp2.id AS id,
    json_agg(jsonb_build_object('scope', tmp2."scope")) AS scope_details
  FROM (
		SELECT id, jsonb_array_elements(scopes::JSONB) AS "scope" FROM auth_consumer WHERE auth_consumer.scopes::TEXT <> 'null'
	) AS tmp2 GROUP BY tmp2.id
) AS tmp1
WHERE
  auth_consumer.id = tmp1.id AND auth_consumer.scopes::TEXT <> 'null';

-- +migrate Down
ALTER TABLE "auth_consumer" DROP COLUMN scope_details;
