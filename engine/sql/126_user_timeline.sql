-- +migrate Up
select create_primary_key('user_timeline', 'user_id');

-- +migrate Down

select 1;
