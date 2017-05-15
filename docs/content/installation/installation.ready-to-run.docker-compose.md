+++
title = "Docker Compose"

[menu.main]
parent = "ready-to-run"
identifier = "docker-compose"

+++

## Run with Docker-Compose

The [docker-compose.yml](https://github.com/ovh/cds/blob/master/docker-compose.yml) contains:

- cds-db service with a postgresql
- cds-cache service with a redis
- cds-migrate service to prepare DB tables
- cds-api service
- cds-ui service
- cds-hatchery-swarm service
- cds-hatchery-local service

Docker compose is very convenient to launch CDS for testing it. But this is not recommended for a Production Installation.

## How to run

```bash
$ git clone https://github.com/ovh/cds.git
$ cd cds
$ export HOSTNAME=$(hostname)

# Create PG Database
$ docker-compose up --no-recreate -d cds-db

# check if db is UP
# check if last log is "LOG:  database system is ready to accept connections"
$ docker-compose logs

$ docker-compose up --no-recreate cds-migrate
# You should have this log: "cds_cds-migrate_1 exited with code 0"

# run API and UI
$ docker-compose up cds-api cds-ui

```

Open a browser on http://localhost:2015, then register a new user.
As there is no SMTP server configured in docker-compose.yml file,
run `docker-compose logs` to get URL for validate the registration.

## Prepare Project, Pipeline and Application

On UI http://localhost:2015:

- Create a project
- Create an application, with a void pipeline
- Create a pipeline, attached to application
- On Pipeline, add a stage and a job
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

Run pipeline. As you can see now, your pipeline is in "waiting status". You have
to run a CDS Worker or a CDS Hatchery which aims to create workers.

Let's run a hatchery with docker-compose.

We will spawn a container with a hatchery in `local` mode. Workers will be spawn inside this container.

```bash
$ docker-compose up cds-hatchery-local
```

*Running a hatchery "local" in a container is not recommended. Use this way only for test purpose*.

After running this Hatchery, a worker will be spawned. Your pipeline will be in "Building", then "Success" status.

## Hatchery Docker Swarm

The [docker-compose.yml](https://github.com/ovh/cds/blob/master/docker-compose.yml) runs hatchery belonging to the `shared.infra` groups.

A `local hatchery` spawns workers on the same host as the hatchery. This is usually useful for specific cases, as
running job on specific hardware.
But this hatchery does not make it possible to respect the isolation of workpaces
of workers as they share the same workspace.

There is another hatchery configured in [docker-compose.yml](https://github.com/ovh/cds/blob/master/docker-compose.yml) file: a 'swarm hatchery'

Please check that your docker installation allows docker API calls on `tcp://${HOSTNAME}:2375`
Otherwise, please update environment variable `DOCKER_HOST: tcp://${HOSTNAME}:2375` in
[docker-compose.yml](https://github.com/ovh/cds/blob/master/docker-compose.yml)

```bash
$ export HOSTNAME=$(hostname)
$ docker-compose up cds-hatchery-swarm
```

A `swarm hatchery` spawns CDS Workers inside dedicated containers.
This ensures isolation of the workspaces and resources.

## Next with Actions, Plugins and Templates

- You can download CDS CLI from https://github.com/ovh/cds/releases
- Run:
```bash
$ mv cds-linux-amd64 cds
$ chmod +x cds
$ ./cds login
# enter: http://${HOSTNAME}:8081 as CDS Endpoint
```

- Import actions, example:
```bash
$ cds action add --url https://raw.githubusercontent.com/ovh/cds/master/contrib/actions/cds-docker-package.hcl
```

- Import plugins, example:
```bash
# download plugin-download-linux-amd64 from  https://github.com/ovh/cds/releases
$ cds admin plugin add ./plugin-download-linux-amd64
```

- Import templates, example:
```bash
# download cds-template-plain-linux-amd64 from  https://github.com/ovh/cds/releases
$ cds admin templates add ./cds-template-plain-linux-amd64
```

# Go further

- First pipeline with CDS CLI [read more]({{< relref "getting-started.first-pipeline-ui.md" >}})
- How to use Openstack infrastructure to spawn CDS container [read more]({{< relref "advanced.hatcheries.openstack.md" >}})
- Link CDS to a repository manager, as Github or Bitbucket [read more]({{< relref "advanced.repositories_manager.md" >}})
- Learn more about CDS variables [read more]({{< relref "building-pipelines.variables.md" >}})
