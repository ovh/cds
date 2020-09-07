#!/bin/bash

PGUSER=$1
PGPASSWORD=$2
PGNAME=$3
PGHOST=$4
PGPORT=$5

PGUSER=${PGUSER:-cds}
PGPASSWORD=${PGPASSWORD:-cds}
PGNAME=${PGNAME:-cds}
PGHOST=${PGHOST:-localhost}
PGPORT=${PGPORT:-5432}
export PGUSER PGPASSWORD PGHOST PGPORT

cat << EOF > missing_pk.sql
with cte as (
     select n.nspname as schema,
            c.relname as "table",
            array_agg(i.indisprimary) as indexes
       from pg_class c
  left join pg_index i on c.oid = i.indrelid
       join pg_namespace n on c.relnamespace = n.oid
      where c.relkind = 'r'
        and n.nspname not in ('pg_catalog', 'information_schema')
   group by "table", schema
)
select quote_ident(schema) || '.' || quote_ident("table") as "table"
    from cte
    where not indexes @> ARRAY[true];
EOF

NC='\033[0m' # No Color
RED='\033[0;31m'
GREEN='\033[0;32m'

return_code=0
echo "Checking missing primary key"
echo "  Hostname: $PGHOST:$PGPORT"
echo "  User: $PGUSER"
echo "  Password: **********"
echo "  Database: $PGNAME"

echo "======================================================================"
psql -t -f missing_pk.sql -o missing_pk.log -h $PGHOST -d $PGNAME -U $PGUSER
if grep -qvE '^\s*$' missing_pk.log
then
    echo "SM0005: Missing primary keys"
    cat missing_pk.log
    return_code=1
else
    echo -e "${GREEN}OK${NC}"
fi
echo "======================================================================"

rm -f missing_pk.sql

exit $return_code
