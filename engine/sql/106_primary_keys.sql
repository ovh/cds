-- +migrate Up
 
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION create_primary_key(tablename text, column_names text) RETURNS void AS $$
declare
   l_count integer;
begin
  select count(*) into l_count from information_schema.table_constraints where table_name = lower(tablename) and constraint_type = 'PRIMARY KEY';
  if l_count = 0 then
     execute 'ALTER TABLE ' || tablename || ' ADD PRIMARY KEY (' || array_to_string(string_to_array(column_names, ',') , ',') || ')';
  end if;
end;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate Down

select 1;
