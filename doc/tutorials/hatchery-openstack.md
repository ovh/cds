# CDS build using OVH.com Openstack infrastructure

## Create Openstack user

In OVH manager, in [cloud section](https://www.ovh.com/manager/cloud), click on the menu on the *Servers>Openstack* item.

You will be able to create an Openstack user, enter description (name and password will be generated).

## Add Openstack worker model

We need to define an Openstack worker model to have Openstack hatchery booting workers.

We will create a model called docker:

 * With low hardware capacity (vps-ssd-1)
 * On Debian 8
 * With docker ready to use
 * Git installed

First, define a udata file. It is a shell script executed just after the boot sequence complete. Our docker udata will look like this:

```shell
# Install docker
cd $HOME
sudo apt-get -y --force-yes update >> /tmp/user_data 2>&1
apt-get install -y --force-yes apt-transport-https ca-certificates >> /tmp/user_data 2>&1
apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
sudo mkdir -p /etc/apt/sources.list.d
sudo sh -c "echo deb https://apt.dockerproject.org/repo debian-jessie main > /etc/apt/sources.list.d/docker.list"
sudo apt-get -y --force-yes update >> /tmp/user_data 2>&1
sudo apt-cache policy docker-engine >> /tmp/user_data 2>&1
sudo apt-get install -y --force-yes docker-engine >> /tmp/user_data 2>&1
sudo service docker start >> /tmp/user_data 2>&1

# Non-root access
sudo groupadd docker >> /tmp/user_data 2>&1
sudo gpasswd -a ${USER} docker >> /tmp/user_data 2>&1
sudo service docker restart >> /tmp/user_data 2>&1

# Basic build binaries
sudo apt-get -y --force-yes install curl git >> /tmp/user_data 2>&1
sudo apt-get -y --force-yes install binutils >> /tmp/user_data 2>&1
```

Last step, define worker model in cds:

```shell
$ cds worker model add docker openstack --image="Debian 8" --flavor="vps-ssd-1" --userdata="./docker.udata"
```

Declare docker and git capabilities
``` shell
$ cds worker model capability add docker docker binary docker
$ cds worker model capability add docker git binary git
```

# Start Opentack hatchery

Generate a token for group:

```shell
$ cds generate  token -g shared.infra -e persistent
fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

Then start hatchery:

```shell
OPENSTACK_USER=<user> OPENSTACK_TENANT=<tenant> OPENSTACK_AUTH_ENDPOINT=https://auth.cloud.ovh.net OPENSTACK_PASSWORD=<password> OPENSTACK_REGION=SBG1 hatchery openstack \
        --api=https://api.domain \
        --max-worker=10 \
        --provision=1 \
        --token=fc300aad48242d19e782a37d361dfa3e55868a629e52d7f6825c7ce65a72bf92
```

This hatchery will now start worker of model 'docker' on OVH.com Openstack infrastructure when a pipeline is in queue with requirement docker.
