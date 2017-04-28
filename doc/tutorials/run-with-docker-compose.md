## Installation

The [docker-compose.yml](/docker-compose.yml) contains:
- cds-db service with a postgresql
- cds-cache service with a redis
- cds-migrate service to prepare DB tables.
- cds-api service
- cds-ui service
- cds-hatchery-swarm service
- cds-hatchery-local service

Docker compose is very convenient to launch CDS for testing it. But this is not recommended for a Production Installation.

## How to run

```bash
$ git clone https://github.com/ovh/cds.git
cd cds
export HOSTNAME=$(hostname)

# Create PG Database
docker-compose up --no-recreate -d cds-db

# check if db is UP
# check if last log is "LOG:  database system is ready to accept connections"
docker-compose logs cds_cds-db_1

docker-compose up --no-recreate cds-migrate
# You should have this log: "cds_cds-migrate_1 exited with code 0"

# run last API, UI and Hatchery
docker-compose up cds-api cds-ui

```

Open a browser on http://localhost:2015, then register a new user.
As there is no SMTP server configured in docker-compose.yml file,
run `docker-compose logs` to get URL for validate the registration.

## Prepare Project, Pipeline and Application

On UI:

- Create a project
- Create an application, with an void pipeline
- Create a pipeline, with a stage and a job
- Inside job, add a step of type "script"
- In script content, add theses lines:
```bash
#!/bin/bash
set -ex
echo "foo"
sleep 10
echo "bar"
```

## Run Pipeline

Run pipeline. As you can see now, you pipeline is in "waiting status". You have
to run a CDS Worker or a CDS Hatchery which aims to create workers.

Let's run an hatchery with docker-compose. Two ways:
- a containers with a hatchery "local". Workers will be spawn inside this container.
- a containers with a hatchery "swarm". Each workerswill be in their own container.

If your host expose docker API, you can run `docker-compose up cds-hatchery-swarm`
Otherwise, you can run `docker-compose up cds-hatchery-local`

*Running a hatchery "local" in a container is not recommanded. Use this way only for test purpose*.

After running a Hatchery, your pipeline will be in "Building" status, then "Success" status.
