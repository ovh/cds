# Venom - Executor Database Fixtures

Step to load fixtures into **MySQL** and **PostgreSQL** databases.

It use the package `testfixtures.v2` under the hood: https://github.com/go-testfixtures/testfixtures
Please read its documentation for further details about the parameters of this executor, especially `folder` and `files`, and how you should write the fixtures.

## Input
In your yaml file, you declare tour step like this

```yaml
  - database mandatory [mysql/postgres]
  - dsn mandatory
  - schemas optional
  - files optional
  - folder optional
 ```

- `schemas` is a list of paths to several `.sql` file that contains the schemas of the tables in your database. If specified, the content of every file will be executed before loading the fixtures.
- `files` parameter is only used as a fallback if `folder` is not used.

Example usage (_mysql_):
```yaml

name: Title of TestSuite
testcases:

  - name: Load database fixtures
    steps:
      - type: dbfixtures
        database: mysql
        dsn: user:password@(localhost:3306)/venom?multiStatements=true
        schemas:
          - schemas/mysql.sql
        folder: fixtures
        files:
          - fixtures/table.yml

```

*note: in the example above, the query param `multiStatements=true` is mandatory if we want to be able to load the schema.*

## SQL drivers

This executor uses the following SQL drivers:

- _MySQL_: https://github.com/go-sql-driver/mysql
- _PostgreSQL_: https://github.com/lib/pq
