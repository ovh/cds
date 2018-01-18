-- +migrate Up
ALTER TABLE application ADD COLUMN vcs_strategy JSONB;
UPDATE action_parameter SET type = 'string' where type = 'key';

DELETE from action_parameter where name = 'authPrivateKey' AND action_id = (select id from action where name = 'GitTag');
DELETE from action_parameter where name = 'signKey' AND action_id = (select id from action where name = 'GitTag');
DELETE from action_parameter where name = 'url' AND action_id = (select id from action where name = 'GitTag');
DELETE from action_parameter where name = 'user' AND action_id = (select id from action where name = 'GitTag');
DELETE from action_parameter where name = 'password' AND action_id = (select id from action where name = 'GitTag');
DELETE from action_parameter where name = 'username' AND action_id = (select id from action where name = 'GitTag');
DELETE from action_parameter where name = 'userEmail' AND action_id = (select id from action where name = 'GitTag');


-- +migrate Down
ALTER TABLE application DROP COLUMN vcs_strategy;


