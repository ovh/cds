---
title: "Docker Compose"
weight: 1
card: 
  name: ready-to-run
---

## Run with Docker-Compose

The [docker-compose.yml](https://github.com/ovh/cds/blob/master/docker-compose.yml) contains:

- cds-db service with a PostgreSQL
- cds-cache service with a Redis
- elasticsearch service with an Elasticsearch
- cds-migrate service to prepare DB tables
- cds-api service
- cds-ui service
- cds-elasticsearch service
- cds-hatchery-swarm service
- cds-hatchery-local service
- cds-vcs service
- cds-repositories service

Docker compose is very convenient to launch CDS for testing it. But this is not recommended for a Production Installation.

## How to run

```bash
$ mkdir /tmp/cdstest && cd /tmp/cdstest
$ curl https://raw.githubusercontent.com/ovh/cds/master/docker-compose.yml -o docker-compose.yml
$ export HOSTNAME=$(hostname)

# Get the latest version
$ docker pull ovhcom/cds-engine:latest

# Create PG database and cache
$ docker-compose up --no-recreate -d cds-db cds-cache elasticsearch

# check if DB is up
# check if last log is "LOG: database system is ready to accept connections"
$ docker-compose logs

$ docker-compose up --no-recreate cds-migrate
# You should have this log: "cds_cds-migrate_1 exited with code 0"

# prepare initial configuration.
$ docker-compose up cds-prepare

# run API, UI and hooks µservice
$ docker-compose up -d cds-api

# the INIT_TOKEN variable will be used by cdsctl to create first admin user
$ TOKEN_CMD=$(docker logs test_cds-prepare_1|grep TOKEN)

# then execute the export INIT_TOKEN
$ $TOKEN_CMD

# create user
$ curl http://localhost:8081/download/cdsctl/linux/amd64\?variant=nokeychain -o cdsctl
$ chmod +x cdsctl
$ ./cdsctl signup --api-url http://localhost:8081 --email admin@localhost.local --username admin --fullname admin
# enter a strong password

# verify the user
$ docker-compose logs cds-api|grep 'cdsctl signup verify'

# run the command find in the output logs
$ ./cdsctl signup verify --api-url http://localhost:8081 very-very-long-token-here

# run cdsctl 
$ ./cdsctl user me

# should returns something like:
#./cdsctl user me
#created   2019-12-18 14:25:53.089718 +0000 UTC
#fullname  admin
#id        vvvvv-dddd-eeee-dddd-fffffffff
#ring      ADMIN
#username  admin

# run others services
$ docker-compose up -d cds-ui cds-hooks cds-hatchery-local cds-elasticsearch
```

- Login on WebUI

Open a browser on http://localhost:8080/account/signup, then login with the user `admin`,

- Import a workflow template

```bash
$ ./cdsctl template push https://raw.githubusercontent.com/ovh/cds/master/contrib/workflow-templates/demo-workflow-hello-world/demo-workflow-hello-world.yml
Workflow template shared.infra/demo-workflow-hello-world has been created
Template successfully pushed !
```

- Create a project, then create your first workflow

```bash
$ ./cdsctl project create DEMO FirstProject
$ ./cdsctl workflow applyTemplate
? Found one CDS project DEMO - FirstProject. Is it correct? Yes
? Choose the CDS template to apply: Demo workflow hello world (shared.infra/demo-workflow-hello-world)
? Give a valid name for the new generated workflow MyFirstWorkflow
? Push the generated workflow to the DEMO project Yes
Application MyFirstWorkflow successfully created
Application variables for MyFirstWorkflow are successfully created
Permission applied to group FirstProject to application MyFirstWorkflow
Environment MyFirstWorkflow-prod successfully created
Environment MyFirstWorkflow-dev successfully created
Environment MyFirstWorkflow-preprod successfully created
Pipeline build-1 successfully created
Pipeline deploy-1 successfully created
Pipeline it-1 successfully created
Workflow MyFirstWorkflow has been created
Workflow successfully pushed !
.cds/MyFirstWorkflow.yml
.cds/build-1.pip.yml
.cds/deploy-1.pip.yml
.cds/it-1.pip.yml
.cds/MyFirstWorkflow.app.yml
.cds/MyFirstWorkflow-dev.env.yml
.cds/MyFirstWorkflow-preprod.env.yml
.cds/MyFirstWorkflow-prod.env.yml
```

- Run CDS Workflow!

```bash
$ ./cdsctl workflow run DEMO MyFirstWorkflow
Workflow MyFirstWorkflow #1 has been launched
http://localhost:8080/project/DEMO/workflow/MyFirstWorkflow/run/1
```

- Check on UI

on http://localhost:8080/project/DEMO/workflow/MyFirstWorkflow/run/1 you will have

![Workflow Generated](/images/ready_to_run_docker_compose_ui.png)

You see that the pipeline deploy in production was not launched automatically. 
There is a Run Condition on it `cds.manual = true`: 

![Run Condition](/images/ready_to_run_docker_compose_run_condition.png)

The build pipeline contains two stages, with only one job in each stage

![Build Pipeline](/images/ready_to_run_docker_compose_build_pipeline.png)

## Next with Swarm Hatchery

- Create our first worker model

The previous workflow was launched with the Hatchery Local. This hatchery is only for dev purpose, we will
now create a Docker worker model and run the Hatchery Docker Swarm.

```bash
$ ./cdsctl worker model import https://raw.githubusercontent.com/ovh/cds/master/contrib/worker-models/go-official-1.11.4-stretch.yml
Worker model go-official-1.11.4-stretch imported with success
```

- Hatchery Docker Swarm

The [docker-compose.yml](https://github.com/ovh/cds/blob/master/docker-compose.yml) runs hatchery belonging to the `shared.infra` groups.

Please check that your Docker installation allows Docker API calls on `tcp://${HOSTNAME}:2375`
Otherwise, please update environment variable `DOCKER_HOST: tcp://${HOSTNAME}:2375` in
[docker-compose.yml](https://github.com/ovh/cds/blob/master/docker-compose.yml)

```bash
$ export HOSTNAME=$(hostname)
$ # For osX user run this container. This will allow hatchery:swarm to communicate with your docker daemon
$ docker run -d -v /var/run/docker.sock:/var/run/docker.sock -p 2375:2375 bobrik/socat TCP4-LISTEN:2375,fork,reuseaddr UNIX-CONNECT:/var/run/docker.sock
$ docker-compose up -d cds-hatchery-swarm
```

A `swarm hatchery` spawns CDS Workers inside dedicated containers.
This ensures isolation of the workspaces and resources.

## Setup connection with a VCS

```bash
# READ THE section https://ovh.github.io/cds/docs/integrations/github/#create-a-cds-application-on-github to generate the clientId and clientSecret.
# Short version: 
# go on https://github.com/settings/applications/new
# Application name: cds-test-docker-compose
# Homepage URL: http://localhost:8080
# Authorization callback: http://localhost:8080/cdsapi/repositories_manager/oauth2/callback
# send click on register application.
$ export CDS_EDIT_CONFIG="vcs.servers.github.github.clientId=xxxxx vcs.servers.github.github.clientSecret=xxxxx " 
$ docker-compose up cds-edit-config
$ docker-compose up -d cds-vcs cds-repositories
```

Notice that here, you have the VCS and Repositories services up and running.

*vcs*: The aim of this µService is to communicate with Repository Manager as GitHub, GitLab, Bitbucket…
But, as your CDS is not probably public, GitHub won't be able to call your CDS to automatically run your workflow on each git push.

*repositories*: this µService is used to enable the as-code feature.
Users can store CDS Files on their repositories. This service clones user repositories on local filesystem.


## Then, next with Actions, Plugins

- Import actions, example:

```bash
$ ./cdsctl action import https://raw.githubusercontent.com/ovh/cds/master/contrib/actions/cds-docker-package.yml
```

## Go further

- How to use OpenStack infrastructure to spawn CDS Workers [read more]({{< relref "/docs/integrations/openstack/openstack_compute.md" >}})
- Link CDS to a repository manager, as [GitHub]({{< relref "/docs/integrations/github/_index.md" >}}), [Bitbucket Server]({{< relref "/docs/integrations/bitbucket.md" >}}) or [GitLab]({{< relref "/docs/integrations/gitlab/_index.md" >}})
- Learn more about CDS variables [read more]({{< relref "/docs/concepts/variables.md" >}})
