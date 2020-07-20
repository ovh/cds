-- +migrate Up
UPDATE action set description = 'CDS Builtin Action. Tag the current branch and push it. Semver used if fully compatible with https://semver.org/' where name = 'GitTag';
UPDATE action_parameter SET name = 'tagMetadata', description = 'Metadata of the tag. Example: cds.42 on a tag 1.0.0 will return 1.0.0+cds.42' where name = 'tagName' and action_id = (select id from action where name = 'GitTag');
INSERT into action_parameter (action_id, name, description, type, value) values((select id from action where name = 'GitTag'), 'tagPrerelease', 'Prerelase version of the tag. Example: alpha on a tag 1.0.0 will return 1.0.0-apha', 'string', '');
INSERT into action_parameter (action_id, name, description, type, value) values((select id from action where name = 'GitTag'), 'tagLevel', 'Set the level of the tag. Must be ''major'' or ''minor'' or ''patch''', 'string', '');


-- +migrate Down
UPDATE action_parameter set name = 'tagName', description = 'Set the name of the tag. Must match semver. If empty CDS will make a patch version' where name = 'tagMetadata' and action_id = (select id from action where name = 'GitTag');
DELETE from action_parameter where name = 'tagPrerelease' and action_id = (select id from action where name = 'GitTag');
DELETE from action_parameter where name = 'tagLevel' and action_id = (select id from action where name = 'GitTag');
