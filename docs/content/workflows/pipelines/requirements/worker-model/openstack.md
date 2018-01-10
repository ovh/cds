+++
title = "Setup Openstack Worker Model"
weight = 4

+++

CDS build using OVH.com Openstack infrastructure

## Create Openstack user

In OVH manager, in [cloud section](https://www.ovh.com/manager/cloud), click on the menu on the *Servers>Openstack* item.

You will be able to create a worker model Openstack user, enter description (name and password will be generated).

## Add Openstack worker model

We need to define an Openstack worker model to have Openstack hatchery booting workers.

We will create a model called docker:

 * With low hardware capacity (vps-ssd-1)
 * On Debian 8
 * With docker ready to use
 * Git installed

First, define a udata file. It is a shell script executed just after the boot sequence complete. Our docker udata will look like this:

```bash
# Install docker
cd $HOME
apt-get -y --force-yes update >> /tmp/user_data 2>&1
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

# Basic build binaries
apt-get -y --force-yes install curl git >> /tmp/user_data 2>&1
apt-get -y --force-yes install binutils >> /tmp/user_data 2>&1
```

Last step, define worker model in cds:

```bash
$ cds worker model add docker openstack --image="Debian 8" --flavor="vps-ssd-1" --userdata="./docker.udata"
```
