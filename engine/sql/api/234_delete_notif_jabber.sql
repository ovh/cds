-- +migrate Up
delete from workflow_notification where type = 'jabber';

-- +migrate Down
SELECT 1;
