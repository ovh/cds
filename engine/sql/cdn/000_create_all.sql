-- +migrate Up
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION create_index(table_name text, index_name text, column_name text) RETURNS void AS $$
declare
   l_count integer;
begin
  select count(*)
     into l_count
  from pg_indexes
  where schemaname = 'public'
    and tablename = lower(table_name)
    and indexname = lower(index_name);

  if l_count = 0 then
     execute 'create index ' || index_name || ' on "' || table_name || '"(' || column_name || ')';
  end if;
end;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION create_unique_index(table_name text, index_name text, column_names text) RETURNS void AS $$
declare
   l_count integer;
begin
  select count(*)
     into l_count
  from pg_indexes
  where schemaname = 'public'
    and tablename = lower(table_name)
    and indexname = lower(index_name);

  if l_count = 0 then
     execute 'create unique index ' || index_name || ' on "' || table_name || '"(' || array_to_string(string_to_array(column_names, ',') , ',') || ')';
  end if;
end;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION create_foreign_key(fk_name text, table_name_child text, table_name_parent text, column_name_child text, column_name_parent text) RETURNS void AS $$
declare
   l_count integer;
begin
  select count(*)
     into l_count
  from information_schema.table_constraints as tc
  where constraint_type = 'FOREIGN KEY'
    and tc.table_name = lower(table_name_child)
    and tc.constraint_name = lower(fk_name);

  if l_count = 0 then
     execute 'alter table "' || table_name_child || '" ADD CONSTRAINT ' || fk_name || ' FOREIGN KEY(' || column_name_child || ') REFERENCES "' || table_name_parent || '"(' || column_name_parent || ')';
  end if;
end;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION create_foreign_key_idx_cascade(fk_name text, table_name_child text, table_name_parent text, column_name_child text, column_name_parent text) RETURNS void AS $$
declare
   l_count integer;
begin
  select count(*)
     into l_count
  from information_schema.table_constraints as tc
  where constraint_type = 'FOREIGN KEY'
    and tc.table_name = lower(table_name_child)
    and tc.constraint_name = lower(fk_name);

  if l_count = 0 then
     execute 'alter table "' || table_name_child || '" ADD CONSTRAINT ' || fk_name || ' FOREIGN KEY(' || column_name_child || ') REFERENCES "' || table_name_parent || '"(' || column_name_parent || ') ON DELETE CASCADE';
     execute create_index(table_name_child, 'IDX_' || fk_name, column_name_child);
  end if;
end;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate Down
-- nothing to downgrade, it's a creation !
