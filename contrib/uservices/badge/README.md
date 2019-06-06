# Badge CDS Service [Under active development]

µService which generate badge for CDS workflow. 

## Prerequisites

+ PostgreSQL
+ Kafka (using the SASL authentication. Or nothing for sandboxing)
+ [Diesel cli](https://github.com/diesel-rs/diesel/tree/master/diesel_cli)

## Usage

There are 2 different modes.

+ `kafka` : the µService will simply listen to the kafka topic of CDS events. 
+ `web` : the µService act like a CDS service and register to the API. It needs authentified HTTP API calls to save states of workflows.

```bash
$ export DATABASE_URL=postgres://username:password@hostname/table_name # Only useful for diesel cli
$ diesel migration run # Initialize database and make migrations
$ ./badge-cds-service config new # You have to edit the config.toml file generated to correspond with your configuration before next command
$ ./badge-cds-service start # You can indicate a -f path_to_my_conf_file.toml
```

## TODO

- [ ] add some tests

## Development

Use cargo to compile.

```bash
$ cargo build # or cargo run
```
