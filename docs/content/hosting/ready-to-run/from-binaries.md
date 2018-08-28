+++
title = "Run with binaries"
weight = 2

+++

## Run from binaries

This article contains the steps to start CDS locally, with API, UI and a local Hatchery.

- Download CDS from Github
- Prepare Database
- Launch CDS API, CDS UI and a Local Hatchery

### Prerequisite

- a redis
- a postgresql 9.4 min

### Get latest release from Github

```bash
mkdir $HOME/cds
cd cds

LAST_RELEASE=$(curl -s https://api.github.com/repos/ovh/cds/releases | grep tag_name | head -n 1 | cut -d '"' -f 4)
OS=linux # could be linux, darwin, windows
ARCH=amd64 # could be 386, arm, amd64

# GET Binaries from github
wget https://github.com/ovh/cds/releases/download/$LAST_RELEASE/cds-engine-$OS-$ARCH
wget https://github.com/ovh/cds/releases/download/$LAST_RELEASE/cds-worker-$OS-$ARCH
wget https://github.com/ovh/cds/releases/download/$LAST_RELEASE/cdsctl-$OS-$ARCH
wget https://github.com/ovh/cds/releases/download/$LAST_RELEASE/ui.tar.gz
wget https://github.com/ovh/cds/releases/download/$LAST_RELEASE/sql.tar.gz
chmod +x *-$OS-$ARCH

```

### Prepare Database

For this example, we consider that the database is installed on `localhost`, 
port `5432`, with an existing empty database and user named `cds` and a password 'cds'.

```bash
cd $HOME/cds
tar xzf sql.tar.gz
./cds-engine-linux-amd64 database upgrade --db-host localhost --db-user cds --db-password cds --db-name cds --db-sslmode disable --db-port 5432 --migrate-dir sql
```

### Launch CDS API

Generate a **[Configuration File]({{<relref "/hosting/configuration/_index.md" >}})**

```bash
cd $HOME/cds

./cds-engine-linux-amd64 config new > $HOME/cds/conf.toml
./cds-engine-linux-amd64 start api --config $HOME/cds/conf.toml
```

Check that CDS is up and running:

```bash
curl http://localhost:8081/mon/version
curl http://localhost:8081/mon/status
```

### Launch CDS UI

```bash
cd $HOME/cds
tar xzf ui.tar.gz # this command generates a $HOME/cds/dist/ directory
```

The `dist/` directory contains all HTML, Javascript, css... files.

You can serve theses files with a simple web server, but there is a ready-to-run Caddyfile to launch CDS UI quickly.

```bash
cd dist/

# BACKEND_HOST contains a url to CDS Engine
export BACKEND_HOST="http://localhost:8081"

# if you expose CDS on a domain as https://your-domain/your-cds, enter "/your-cds"
BASE_URL="/"
sed -i "s#base href=\"/\"#base href=\"${BASE_URL}\"#g" index.html

# Get Caddy
wget https://github.com/ovh/cds/releases/download/0.8.0/caddy-linux-amd64 
chmod +x caddy-linux-amd64 

# RUN CDS UI
./caddy-linux-amd64 
```

Then, open a browser on http://localhost:2015/ . You have to signup your first CDS user. It will be an administrator on CDS. In order to do that, just go on UI and click on signup or use `cdsctl signup`. If you don't have email service configured you just have to check your CDS API logs to have the confirmation link.

### Launch CDS Local Hatchery

The previously generated configuration file contains all CDS configuration.

To be able to start a local hatchery, enter a hatchery name in the section `hatchery.local.commonConfiguration`

```toml

...
[hatchery.local]

    # BaseDir for worker workspace
    basedir = "/tmp"

    # Nb Workers to provision
    nbProvision = 1

    [hatchery.local.commonConfiguration]

      # Name of Hatchery
      name = "my-local-hatchery"
...

```

Then, start the local hatchery


```bash
./cds-engine-linux-amd64 start hatchery:local --config $HOME/cds/conf.toml

# notice that you can run api and hatchery with a one common only:
# ./cds-engine-linux-amd64 start api hatchery:local --config $HOME/cds/conf.toml
```

# Go further

- How to use Openstack infrastructure to spawn CDS container [read more]({{< relref "hatchery/openstack.md" >}})
- Link CDS to a repository manager, as Github, Bitbucket Server or Gitlab [read more]({{< relref "/hosting/repositories-manager/_index.md" >}})
- Learn more about CDS variables [read more]({{< relref "workflows/pipelines/variables.md" >}})
