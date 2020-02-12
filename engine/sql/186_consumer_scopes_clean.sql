-- +migrate Up
ALTER TABLE "auth_consumer" DROP COLUMN scopes;

-- +migrate Down
ALTER TABLE "auth_consumer" ADD COLUMN scopes JSONB;

UPDATE auth_consumer SET scopes = scope_details WHERE scope_details::TEXT = 'null';

UPDATE auth_consumer
SET scopes = tmp1.scopes
FROM (
	SELECT
    tmp2.id AS id,
    json_agg(tmp2."scope_detail"->>'scope') AS scopes
  FROM (
		SELECT id, jsonb_array_elements(scope_details::JSONB) AS "scope_detail" FROM auth_consumer WHERE auth_consumer.scope_details::TEXT <> 'null'
	) AS tmp2 GROUP BY tmp2.id
) AS tmp1
WHERE
  auth_consumer.id = tmp1.id AND auth_consumer.scope_details::TEXT <> 'null';
