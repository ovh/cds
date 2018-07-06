-- +migrate Up
ALTER TABLE project ADD COLUMN description TEXT DEFAULT '';
ALTER TABLE project ADD COLUMN icon TEXT DEFAULT '';
ALTER TABLE application ADD COLUMN icon TEXT DEFAULT '';
ALTER TABLE application DROP COLUMN description;
ALTER TABLE application ADD COLUMN description TEXT DEFAULT '';
ALTER TABLE pipeline ADD COLUMN description TEXT DEFAULT '';

-- +migrate Down
ALTER TABLE project DROP COLUMN description;
ALTER TABLE project DROP COLUMN icon;
ALTER TABLE application DROP COLUMN description;
ALTER TABLE application ADD COLUMN description TEXT;
ALTER TABLE application DROP COLUMN icon;
ALTER TABLE pipeline DROP COLUMN description;
