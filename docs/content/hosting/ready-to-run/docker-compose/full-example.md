---
title: "Docker Compose Full Tutorial"
weight: 2
card: 
  name: ready-to-run
---

## Run with Docker-Compose - Full Tutorial

This page will help you to create a public instance of CDS:

 - installed on a Virtual Machine with a Public Cloud Instance on Openstack
 - with a domain name and SSL configured
 - installed with docker-compose

The whole tutorial of [docker-compose]({{<relref "/hosting/ready-to-run/docker-compose/docker-compose.md">}}) is duplicated here. 
This article contains additional details on HAPRoxy, SSL configuration, IP Restriction. 

At the end of this tutorial, you will have a CDS running with all CDS Services and a Swarm Hatchery. This CDS is fully functional with GitHub.
A CDS installed with this tutorial should only be used for demonstration only. Please read [this article]({{<relref "/hosting/ready-to-run/from-binaries.md">}}) for a production installation.


## Create the Virtual Machine with OpenStack

Create an OpenStack project on OVHcloud Public Cloud: https://www.ovh.com/manager/public-cloud/#/pci/projects/onboarding

Export Openstack Variables:

```bash
export OS_AUTH_URL=https://auth.cloud.ovh.net/v3/
export OS_IDENTITY_API_VERSION=3
export OS_TENANT_ID=your-tenant-id
export OS_TENANT_NAME="your-tenant-name"
export OS_USERNAME="your-openstack-username"
export OS_PASSWORD="your-openstack-password"
export OS_REGION_NAME="opentack-region"

# create virtual machine.
openstack server create --flavor b2-15-flex --image "Debian 10" --key-name="your-key-name" --nic net-id=Ext-Net cdsdemo
```

This new virtual machime is attached to the `default` security group. This group should allows ingress for port 22 and port 443 (from your remote IP). It must also allow the port 80 for SSL configuration.

## Install Docker on your VM

```bash
# get server public IP
openstack server list

# connect to the vm with
ssh debian@ip-of-your-virtual-machine

# go to root
sudo su

# then install docker
apt-get update && \
apt-get install -y apt-transport-https ca-certificates software-properties-common curl git netcat make binutils bzip2 gnupg haproxy telnet htop && \
curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -  && \
add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"  && \
apt-get -y update && \
apt-get -y upgrade && \
apt-get install -y --allow-unauthenticated docker-ce docker-ce-cli containerd.io  && \
curl -L https://github.com/docker/compose/releases/download/1.14.0/docker-compose-`uname -s`-`uname -m` > /usr/local/bin/docker-compose && \
chmod +x /usr/local/bin/docker-compose && \
usermod -aG docker debian && \
echo "127.0.0.11 cdsdemo cdsdemo" >> /etc/hosts
```

## Configure SSL

With `root` user:

```bash
export DOMAIN='your-cdsdemo.domain'
export YOUR_MAIL='admin@localhost.local'

apt-get install certbot
certbot certonly --standalone -d ${DOMAIN} --non-interactive --agree-tos --email ${YOUR_MAIL} --http-01-port=80

# then generate pem file
mkdir /etc/haproxy/certs/
cat /etc/letsencrypt/live/$DOMAIN/fullchain.pem /etc/letsencrypt/live/$DOMAIN/privkey.pem > /etc/haproxy/certs/$DOMAIN.pem
chmod -R go-rwx /etc/haproxy/certs

```

## Configure HAProxy

In the content below, replace `your-cdsdemo.domain` by your domain name, then create the file `/etc/haproxy/haproxy.cfg` with the `root` user.

```
global
	log /dev/log	local0
	log /dev/log	local1 notice
	chroot /var/lib/haproxy
	stats socket /run/haproxy/admin.sock mode 660 level admin expose-fd listeners
	stats timeout 30s
	user haproxy
	group haproxy
	daemon

	# Default SSL material locations
	ca-base /etc/ssl/certs
	crt-base /etc/ssl/private

	ssl-default-bind-ciphers ECDH+AESGCM:DH+AESGCM:ECDH+AES256:DH+AES256:ECDH+AES128:DH+AES:RSA+AESGCM:RSA+AES:!aNULL:!MD5:!DSS
	ssl-default-bind-options no-sslv3

defaults
	log	global
	mode	http
	option	httplog
	option	dontlognull
	timeout connect 5000
	timeout client  50000
	timeout server  50000
	errorfile 400 /etc/haproxy/errors/400.http
	errorfile 403 /etc/haproxy/errors/403.http
	errorfile 408 /etc/haproxy/errors/408.http
	errorfile 500 /etc/haproxy/errors/500.http
	errorfile 502 /etc/haproxy/errors/502.http
	errorfile 503 /etc/haproxy/errors/503.http
	errorfile 504 /etc/haproxy/errors/504.http


frontend webstats
	bind your-cdsdemo.domain:9999 ssl crt /etc/haproxy/certs/your-cdsdemo.domain.pem

frontend cdsdemo
	bind your-cdsdemo.domain:443 ssl crt /etc/haproxy/certs/your-cdsdemo.domain.pem
	redirect scheme https if !{ ssl_fc }
	mode http
	default_backend cdsdemo_ui

	# you can enable stats if you want
	# stats enable  # Enable stats page
	# stats hide-version  # Hide HAProxy version
	# stats realm Haproxy\ Statistics  # Title text for popup window
	# stats uri /haproxy_stats  # Stats URI
	# stats auth cds:your-strongpassword # Authentication credentials
	# stats refresh 30s
	# stats show-node

backend cdsdemo_ui
	mode http
	balance roundrobin
	server cdsui 127.0.0.1:8080 check
```

Then restart HAProxy

```bash
service haproxy restart
```

## Register new OAuth Application on GitHub

- go on https://github.com/settings/applications/new
- Application name: `cds-demo`
- Homepage URL: `https://your-cdsdemo.domain`
- Authorization callback: `https://your-cdsdemo.domain/cdsapi/repositories_manager/oauth2/callback`
- Click on `Register application`.

Notice that you can create a new OAuth Application on a GitHub organization:
`https://github.com/organizations/your-organization/settings/applications/new`

You will have the `CientID` and `ClientSecret`

## Install CDS, initialize everything

In the content below, replace the value of

 - CDS_DOMAIN_NAME
 - CDS_GITHUB_CLIENT_ID
 - CDS_GITHUB_CLIENT_SECRET

then create the file `/home/debian/boot.sh` with the user `debian`.

```bash
#!/bin/bash

set -ex

export CDS_DOMAIN_NAME="your-cdsdemo.domain"
export CDS_GITHUB_CLIENT_ID="xxxxxxxxxxx"
export CDS_GITHUB_CLIENT_SECRET="xxxxxxxxxxx"

mkdir -p tools/smtpmock

curl https://raw.githubusercontent.com/ovh/cds/{{< param "version" "master" >}}/docker-compose.yml -o docker-compose.yml
export HOSTNAME=$(hostname)
export CDS_DOCKER_IMAGE=ovhcom/cds-engine:{{< param "version" "latest" >}}

docker pull ovhcom/cds-engine:{{< param "version" "latest" >}}
docker-compose up -d cds-db cds-cache elasticsearch dockerhost
sleep 3
docker-compose logs| grep 'database system is ready to accept connections'
docker-compose up cds-db-init
docker-compose up cds-migrate
sleep 3
docker-compose up cds-prepare
export CDS_EDIT_CONFIG="vcs.servers.github.github.clientId=${CDS_GITHUB_CLIENT_ID} vcs.servers.github.github.clientSecret=${CDS_GITHUB_CLIENT_SECRET} "
docker-compose up cds-edit-config
export CDS_EDIT_CONFIG="api.url.api=http://localhost:8081 api.url.ui=https://${CDS_DOMAIN_NAME} hatchery.swarm.commonConfiguration.api.http.url=http://cds-api:8081"
docker-compose up cds-edit-config
export CDS_EDIT_CONFIG="hatchery.swarm.commonConfiguration.api.http.url=https://${CDS_DOMAIN_NAME}/cdsapi hooks.urlPublic=https://${CDS_DOMAIN_NAME}/cdshooks ui.hooksURL=http://cds-hooks:8083"
docker-compose up cds-edit-config
docker-compose up -d cds-api
sleep 3
TOKEN_CMD=$(docker logs $(docker-compose ps -q cds-prepare) | grep INIT_TOKEN) && $TOKEN_CMD
curl 'http://localhost:8081/download/cdsctl/linux/amd64?variant=nokeychain' -o cdsctl
chmod +x cdsctl
# this line will ask a password for admin user
./cdsctl signup --api-url http://localhost:8081 --email admin@localhost.local --username admin --fullname admin
VERIFY_CMD=$(docker-compose logs cds-api | grep 'cdsctl signup verify' | cut -d '$' -f2 | xargs) && ./$VERIFY_CMD
# this line returns the RING of user, must be ADMIN
./cdsctl user me

export CDS_EDIT_CONFIG="api.url.api=https://${CDS_DOMAIN_NAME}/cdsapi api.url.ui=https://${CDS_DOMAIN_NAME}"
docker-compose up cds-edit-config
docker-compose stop cds-api
docker-compose rm -f cds-api
docker-compose up -d cds-api
sleep 3
docker-compose up -d cds-ui cds-cdn cds-hooks cds-elasticsearch cds-hatchery-swarm cds-vcs cds-repositories
sleep 5

./cdsctl worker model import https://raw.githubusercontent.com/ovh/cds/{{< param "version" "master" >}}/contrib/worker-models/maven3-jdk10-official.yml

./cdsctl template push https://raw.githubusercontent.com/ovh/cds/{{< param "version" "master" >}}/contrib/workflow-templates/demo-workflow-hello-world/demo-workflow-hello-world.yml
./cdsctl project create DEMO FirstProject
./cdsctl template apply DEMO MyFirstWorkflow shared.infra/demo-workflow-hello-world --force --import-push --quiet
./cdsctl workflow run DEMO MyFirstWorkflow

```

With user debian

```
# be sure that you have group docker
groups
# you should have these groups:
# debian adm dialout cdrom floppy sudo audio dip video plugdev netdev docker
# if it's not the case, logout and re-login with debian user.

cd /home/debian
chmod +x boot.sh
./boot.sh
```

The `boot.sh` will ask you the password for `admin` user, you have to enter a strong password.
The script will also ask you the context name for cdsctl, you can choose the default context `default`.

At the end, you should have to log:

```bash
Workflow MyFirstWorkflow #1 has been launched
https://your-cdsdemo.domain/project/DEMO/workflow/MyFirstWorkflow/run/1
```

This url should be accessible at the moment, since your have configured the SSL and haproxy.

The `docker ps` should returns this:

```

$ docker ps
CONTAINER ID        IMAGE                                                 COMMAND                  CREATED             STATUS                    PORTS                      NAMES
02b60d3f98c0        ovhcom/cds-engine:latest                              "sh -c '/app/cds-eng…"   33 seconds ago      Up 32 seconds (healthy)   0.0.0.0:8080->8080/tcp     debian_cds-ui_1
ae8e87c60e2b        ovhcom/cds-engine:latest                              "sh -c '/app/cds-eng…"   35 seconds ago      Up 33 seconds (healthy)                              debian_cds-vcs_1
c2b8852e487a        ovhcom/cds-engine:latest                              "sh -c '/app/cds-eng…"   35 seconds ago      Up 33 seconds (healthy)   127.0.0.1:8083->8083/tcp   debian_cds-hooks_1
fe2fcbee96aa        ovhcom/cds-engine:latest                              "sh -c '/app/cds-eng…"   35 seconds ago      Up 33 seconds (healthy)                              debian_cds-repositories_1
f2eb7b8c4329        ovhcom/cds-engine:latest                              "sh -c '/app/cds-eng…"   35 seconds ago      Up 33 seconds (healthy)                              debian_cds-elasticsearch_1
22dc66a1b2a2        ovhcom/cds-engine:latest                              "sh -c '/app/cds-eng…"   35 seconds ago      Up 33 seconds (healthy)                              debian_cds-hatchery-swarm_1
958ab1703f16        ovhcom/cds-engine:latest                              "sh -c '/app/cds-eng…"   39 seconds ago      Up 39 seconds (healthy)   0.0.0.0:8081->8081/tcp     debian_cds-api_1
9223395500ab        postgres:14.0                                        "docker-entrypoint.s…"   2 minutes ago       Up About a minute         5432/tcp                   debian_cds-db_1
c9b58ce83003        docker.elastic.co/elasticsearch/elasticsearch:6.7.2   "/usr/local/bin/dock…"   2 minutes ago       Up About a minute         9200/tcp, 9300/tcp         debian_elasticsearch_1
08cfe15c3e2c        bobrik/socat                                          "socat TCP4-LISTEN:2…"   2 minutes ago       Up About a minute         127.0.0.1:2375->2375/tcp   debian_dockerhost_1
fc2ac075c000        redis:alpine                                          "docker-entrypoint.s…"   2 minutes ago       Up About a minute         6379/tcp                   debian_cds-cache_1
```

## Tips 

### Limit access to some IP only

Limit access to your current IP:

```bash
# get your the current IP - from your desk
export MY_IP=$(curl ipaddr.ovh)
# Allow your IP to call the 443 port
openstack security group rule create default --protocol tcp --dst-port 443:443 --remote-ip ${MY_IP}/32
# Allow your IP to call the 22 port
openstack security group rule create default --protocol tcp --dst-port 22:22 --remote-ip ${MY_IP}/32

# check if new rules are applied
openstack security group rule list default
```

Allow GitHub to call your CDS

```bash
# Check GitHub IP Hooks on https://api.github.com/meta

# replace $RANGE_GITHUB with the range of GitHub Hooks.
openstack security group rule create default --protocol tcp --dst-port 443:443 --remote-ip $RANGE_GITHUB

# check if new rules are applied
openstack security group rule list default
```

### Disable Signup on you CDS Instance

```
# to run from user debian, from directory /home/debian/
export CDS_EDIT_CONFIG="api.auth.local.signupDisabled=true"
docker-compose up cds-edit-config

# then, restart api
export HOSTNAME=$(hostname)
docker-compose restart cds-api
```

### Reinstall all CDS on the same VM

```sh
# with user debian
# delete all containers and volumes
docker rm -f `docker ps -aq` && docker volume prune

# run boot.sh file
./boot.sh
```
