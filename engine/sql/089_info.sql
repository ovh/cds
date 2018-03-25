-- +migrate Up
CREATE TABLE info (
  id BIGSERIAL PRIMARY KEY,
  title TEXT NOT NULL default '',
  message TEXT NOT NULL default '',
  level TEXT NOT NULL default '',
  created TIMESTAMP WITH TIME ZONE,
  updated TIMESTAMP WITH TIME ZONE,
  archived BOOLEAN default false
);

-- +migrate Down
DROP TABLE info;
