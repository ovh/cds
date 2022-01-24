-- +migrate Up
DELETE from action_edge where child_id = (select id from action where name = 'Serve Static Files' and type = 'Builtin');
DELETE from action where name = 'Serve Static Files' and type = 'Builtin';

-- +migrate Down
select 1;