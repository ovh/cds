+++
title = "Docker Compose"
weight = 1

+++

## Run with Docker-Compose

The [docker-compose.yml](https://github.com/ovh/cds/blob/master/docker-compose.yml) contains:

- cds-db service with a PostgreSQL
- cds-cache service with a Redis
- cds-migrate service to prepare DB tables
- cds-api service
- cds-ui service
- cds-hatchery-swarm service
- cds-hatchery-local service

Docker compose is very convenient to launch CDS for testing it. But this is not recommended for a Production Installation.

## How to run

```bash
$ mkdir /tmp/cdstest && cd /tmp/cdstest
$ curl https://raw.githubusercontent.com/ovh/cds/master/docker-compose.yml -o docker-compose.yml
$ export HOSTNAME=$(hostname)

# Get the latest version
$ docker pull ovhcom/cds-ui:latest
$ docker pull ovhcom/cds-engine:latest

# Create PG database
$ docker-compose up --no-recreate -d cds-db

# check if DB is up
# check if last log is "LOG: database system is ready to accept connections"
$ docker-compose logs

$ docker-compose up --no-recreate cds-migrate
# You should have this log: "cds_cds-migrate_1 exited with code 0"

# run API and UI
$ docker-compose up -d cds-api cds-ui
```

- Create the first user with WebUI

Open a browser on http://localhost:2015/account/signup, then register a new user `admin`,
with an email `admin@localhost.local` for example.
As there is no SMTP server configured in docker-compose.yml file,
run `docker-compose logs` to get URL for validate the registration.

```bash
$ docker-compose logs|grep 'verify/admin'
```

After registration on UI, keep the password displayed, we will use it in next step.

- Login with cdsctl

Please note that the version linux/amd64, darwin/amd64 and windows/amd64 use libsecret / keychain to store the CDS Password.
If you don't want to use the keychain, you can select the version i386.

See: [cdsctl documentation]({{< relref "cli/cdsctl/_index.md" >}})

You can download cdsctl CLI from http://localhost:2015/settings/downloads
```bash
# on a Linux workstation:
$ curl http://localhost:8081/download/cdsctl/linux/amd64 -o cdsctl
# on a osX workstation, it's curl http://localhost:8081/download/cdsctl/darwin/amd64 -o cdsctl
$ chmod +x cdsctl
$ ./cdsctl login --api-url http://localhost:8081 -u admin
CDS API URL: http://localhost:8081
Username: admin
Password:
          You didn't specify config file location; /Users/yourhome/.cdsrc will be used.
Login successful
```

- Test cdsctl

```bash
$ ./cdsctl user me
CDS API:http://localhost:8081
email       admin@localhost.local
username    admin
fullname    Administrator
```

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

Before running your new Workflow, we have to create a worker model and start a Hatchery for spawning workers.

- Create our first worker model

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

- Run CDS Workflow!

```bash
$ ./cdsctl workflow run DEMO MyFirstWorkflow
Workflow MyFirstWorkflow #1 has been launched
http://localhost:2015/project/DEMO/workflow/MyFirstWorkflow/run/1
```

- Check on UI

on http://localhost:2015/project/DEMO/workflow/MyFirstWorkflow/run/1 you will have

![Workflow Generated](/images/ready_to_run_docker_compose_ui.png)

You see that the pipeline deploy in production was not launched automatically. 
There is a Run Condition on it `cds.manual = true`: 

![Run Condition](/images/ready_to_run_docker_compose_run_condition.png)

The build pipeline contains two stages, with only one job in each stage

![Build Pipeline](/images/ready_to_run_docker_compose_build_pipeline.png)

## Next with Actions, Plugins

- Import actions, example:

```bash
$ ./cdsctl action import https://raw.githubusercontent.com/ovh/cds/master/contrib/actions/cds-docker-package.yml
```

- Import plugins: Please read [Plugins]({{< relref "workflows/pipelines/actions/plugins/_index.md" >}})

# Go further

- How to use OpenStack infrastructure to spawn CDS Workers [read more]({{< relref "hatchery/openstack.md" >}})
- Link CDS to a repository manager, as GitHub, Bitbucket Server or GitLab [read more]({{< relref "/hosting/repositories-manager/_index.md" >}})
- Learn more about CDS variables [read more]({{< relref "workflows/pipelines/variables.md" >}})
