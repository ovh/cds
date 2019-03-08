-- +migrate Up

CREATE TABLE IF NOT EXISTS "authentified_user" (
  id VARCHAR(36) PRIMARY KEY,
  username TEXT NOT NULL,
  fullname TEXT NOT NULL,
  email TEXT NOT NULL,
  origin VARCHAR(25) NOT NULL,
  ring VARCHAR(25) NOT NULL
);


CREATE TABLE IF NOT EXISTS "authentified_user_migration" (
    authentified_user_id VARCHAR(36),
    user_id BIGINT,
    PRIMARY KEY (authentified_user_id, user_id)
);

SELECT create_foreign_key('FK_AUTHENTIFIED_USER_MIGRATION_USER', 'authentified_user_migration', 'user', 'user_id', 'id');
SELECT create_foreign_key('FK_AUTHENTIFIED_USER_MIGRATION_AUTHENTIFIED_USER', 'authentified_user_migration', 'authentified_user', 'authentified_user_id', 'id');

-- +migrate Down

DROP TABLE  "authentified_user";
DROP TABLE  "authentified_user_migration";
