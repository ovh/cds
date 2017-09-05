+++
title = "Setup VSphere Worker Model"
weight = 4

[menu.main]
parent = "tutorials"
identifier = "tutorials-vsphere"

+++

CDS build using VSphere infrastructure

## Add VSphere worker model

We need to define a VSphere worker model to have VSphere hatchery booting workers.

We will create a model called debian8-docker:

 * On Debian 8
 * With docker ready to use
 * Git installed

First, define a udata file. It is a shell script executed just after the boot sequence complete. Our udata will look like this:

```bash
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

```bash
# --image is the name of your vm model that you have created before on your host
$ cds worker model add debian8-docker vsphere --image="debian8" --userdata="./yourscript.sh"
```

Declare docker and git capabilities
``` shell
$ cds worker model capability add debian8-docker docker binary docker
$ cds worker model capability add debian8-docker git binary git
```
