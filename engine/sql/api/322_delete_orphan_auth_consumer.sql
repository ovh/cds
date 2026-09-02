-- +migrate Up

-- Deleting a user only removes its auth_consumer_user rows by cascade, the auth_consumer rows are left orphan.
-- Such consumers can't be used anymore, delete them (their sessions and child consumers go by cascade).
DELETE FROM auth_consumer c
WHERE NOT EXISTS (SELECT 1 FROM auth_consumer_user cu WHERE cu.auth_consumer_id = c.id)
  AND NOT EXISTS (SELECT 1 FROM auth_consumer_hatchery ch WHERE ch.auth_consumer_id = c.id);

-- +migrate Down
SELECT 1;
