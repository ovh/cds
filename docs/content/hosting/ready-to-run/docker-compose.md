+++
title = "Docker Compose"
weight = 1

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

Open a browser on http://localhost:2015, then register a new user `admin`,
with a email `admin@localhost.local` for example.
As there is no SMTP server configured in docker-compose.yml file,
run `docker-compose logs` to get URL for validate the registration.

After registration on UI, keep the password displayed, we will use it after.

## Prepare Project, Pipeline and Application

On UI http://localhost:2015:

- Create a project
- Create a workflows
- On the first Pipeline, add a stage and a job
- Inside job, add a step of type "[script]({{< relref "workflows/pipelines/actions/builtin/script.md" >}})"
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

*Running a hatchery "local" in a container is not recommended. Use this way only for testing purpose*.

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

Now, you have to create worker model of type `docker`, please follow [how to create a worker model docker]({{< relref "workflows/pipelines/requirements/worker-model/docker/_index.md" >}}).

## Next with Actions, Plugins

- You can download CDS CLI from https://github.com/ovh/cds/releases or from http://localhost:2015/settings/downloads
```bash
# on a linux workstation:
$ wget http://localhost:2015/cdsapi/download/engine/linux/amd64 -O cdsctl
$ chmod +x cdsctl
$ ./cdsctl login --api-url http://localhost:8081 -u admin
CDS API URL: http://localhost:8081
Username: admin
Password:
          You didn't specify config file location; /Users/yourhome/.cdsrc will be used.
Login successful
```

Please note that the version linux/amd64, darwin/amd64 and windows/amd64 use libsecret / keychain to store the CDS Password.
If you don't want to use the keychain, you can select the version i386.

See: [cdsctl documentation]({{< relref "cli/cdsctl/_index.md" >}})

- Test cdsctl
```bash
$ cdsctl user me
CDS API:http://localhost:8081
fullname  admin
email     admin@localhost.local
username  admin
```

- Import actions, example:
```bash
# get cds-docker-package.yml from https://github.com/ovh/cds/blob/master/contrib/actions/
$ cdsctl action import cds-docker-package.yml
```

- Import plugins: Please read [Plugins]({{< relref "workflows/pipelines/actions/plugins/_index.md" >}})

# Go further

- How to use Openstack infrastructure to spawn CDS container [read more]({{< relref "hatchery/openstack.md" >}})
- Link CDS to a repository manager, as Github, Bitbucket Server or Gitlab [read more]({{< relref "/hosting/repositories-manager/_index.md" >}})
- Learn more about CDS variables [read more]({{< relref "workflows/pipelines/variables.md" >}})
