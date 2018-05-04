+++
title = "Openstack Worker Model"
weight = 3

+++

CDS build using OVH.com Openstack infrastructure

## Create Openstack user

In OVH manager, in [cloud section](https://www.ovh.com/manager/cloud), click on the menu on the *Servers>Openstack* item.

You will be able to create a worker model Openstack user, enter description (name and password will be generated).

## Add Openstack worker model

We need to define an Openstack worker model to have Openstack hatchery booting workers.

We will create a model called testopenstack:

 * With low hardware capacity (vps-ssd-1)
 * On Debian 8
 * With docker ready to use
 * Git installed

You need to configure:

  * The image is your image on which you want to spawn your openstack VM
  * The flavor of your openstack VM
  * If you aren't an administrator you have to choose a configuration pattern in order to fill pre command, worker command and post command with a [pattern that an administrator have already fill for you]({{< relref "workflows/TODO_PATTERNS" >}}).
  * If you are an administrator :
    * pre worker command: all scripts that need to be run before execute the worker binary (for example: set the right environment variables, install curl and other tools you need like docker, ...)
    * main worker command: the command launched to run the worker with right flags thanks to the interpolate variables that CDS fill for you [(more informations click here)]({{< relref "workflows/TODO_VARIABLES" >}}).
    * post worker command: the command launched after the execution of your worker. If you need to clean something and then shutdown the VM.

Via UI (inside settings section --> worker models):

For example:

![Worker Model UI Openstack](/images/worker_model_openstack.png)

Or via cli with a yaml file:

```bash
$ cds worker model import my_worker_model.yml
```


```yaml
name: testopenstack
type: openstack
description: "my worker model"
group: shared.infra
image: "Debian 7"
flavor: vps-ssd-1
pre_cmd: |
  #!/bin/bash
  set +e
  # Basic build binaries
  cd $HOME
  apt-get -y --force-yes update >> /tmp/user_data 2>&1
  apt-get -y --force-yes install curl git >> /tmp/user_data 2>&1
  apt-get -y --force-yes install binutils >> /tmp/user_data 2>&1
  # Docker installation (FOR DEBIAN)
  if [[ "x{{.FromWorkerImage}}" = "xtrue" ]]; then
    echo "$(date) - CDS_FROM_WORKER_IMAGE == true - no install docker required "
  else
    # Install docker
    apt-get install -y --force-yes apt-transport-https ca-certificates >> /tmp/user_data 2>&1
    apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
    mkdir -p /etc/apt/sources.list.d
    sh -c "echo deb https://apt.dockerproject.org/repo debian-jessie main > /etc/apt/sources.list.d/docker.list"
    apt-get -y --force-yes update >> /tmp/user_data 2>&1
    apt-cache policy docker-engine >> /tmp/user_data 2>&1
    apt-get install -y --force-yes docker-engine >> /tmp/user_data 2>&1
    service docker start >> /tmp/user_data 2>&1
    # Non-root access
    groupadd docker >> /tmp/user_data 2>&1
    gpasswd -a ${USER} docker >> /tmp/user_data 2>&1
    service docker restart >> /tmp/user_data 2>&1
  fi;
  curl -L "{{.API}}/download/worker/linux/$(uname -m)" -o worker --retry 10 --retry-max-time 120 -C - >> /tmp/user_data 2>&1
  chmod +x worker

cmd: "./worker --api={{.API}} --token={{.Token}} --basedir={{.BaseDir}} --model={{.Model}} --name={{.Name}} --hatchery={{.Hatchery}} --hatchery-name={{.HatcheryName}} --insecure={{.HTTPInsecure}} --single-use --force-exit"

post_cmd: sudo shutdown -h now

```
