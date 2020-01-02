-- +migrate Up
ALTER TABLE people ADD COLUMN name TEXT;

-- +migrate Down
CREATE TABLE new_people (id int);
INSERT INTO new_people SELECT id FROM people;
DROP TABLE IF EXISTS people; 
ALTER TABLE new_people RENAME TO people;
