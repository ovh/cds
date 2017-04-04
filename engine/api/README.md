# CDS API

This is the core component of CDS.
To start CDS api, the only mandatory dependency is a PostgreSQL database and a path to the directory containing other CDS binaries.

There is are two ways to set up CDS:

- as [https://github.com/toml-lang/toml](toml) configuration
- over environment variables.

## CDS API Third-parties

At the minimum, CDS needs a PostgreSQL Database >= 9.4. But for serious usage your may need :

- A [Redis](https://redis.io) server or sentinels based cluster used as a cache and session store
- A LDAP Server for authentication
- A SMTP Server for mails
- A [Kafka](https://kafka.apache.org/) Broker to manage CDS event
- A [Openstack Swift](https://docs.openstack.org/developer/swift/) Tenant to store builds artifacts
- A [Vault](https://www.vaultproject.io/)server for cipher and app keys

See Configuration template for more details

## Database management

CDS provide all needed tools scripts to perform Schema creation and auto-migration. Those tools are embedded inside the `api` binary.

### Creation

On a brand new database run the following command:

```bash
$ $PATH_TO_CDS/api database upgrade --db-host <host> --db-host <port> --db-password <password> --db-name <database> --limit 0
```

### Upgrade

On an existing database, run the following command on each CDS update:

```bash
$ $PATH_TO_CDS/api database upgrade --db-host <host> --db-host <port> --db-password <password> --db-name <database>
```

### More details

[Read more about CDS Database Management](https://github.com/ovh/cds/tree/master/engine/sql)


## TOML Configuration

The toml configuration can ve provided by a file or via [https://www.consul.io](consul k/v store).

### Start CDS with local configuration file

You can also generate a configuration file template with the following command.

```bash
$ $PATH_TO_CDS/api --config my_conf_file.toml
Generating default config file my_conf_file.toml
```

Edit this file.

Run CDS

```bash
$ $PATH_TO_CDS/api --config my_conf_file.toml
Reading configuration file my_new_file.toml
2017/04/04 16:33:17 [NOTICE]   Starting CDS server...
...
```

### Start CDS with Consul

Upload your `toml` configuration to consul

```bash
$ consul kv put cds/config.api.toml -
<PASTE YOUR CONFIGURATION>
<ENDS WITH CRTL-D>
Success! Data written to: cds/config.api.toml
```

Run CDS

```bash
$ $PATH_TO_CDS/api --remote-config localhost:8500 --remote-config-key cds/config.api.toml
Reading configuration from localhost:8500
2017/04/04 16:11:25 [NOTICE]   Starting CDS server...
...
```

### TOML Configuration template

```toml
###################################
# CDS Configuration file template #
###################################
# Please update this file with your own settings

#####################
# CDS URLs Settings #
#####################
# Set the URLs from the user's point of view. It may be URL of your reverse proxy if you use one.
[url]
api = "http://localhost:8081"
ui = "http://localhost:8080"

#####################
# CDS Logs Settings #
#####################
# Define log levels and hooks
[log]
level = "info"
db = false

# CDS needs local directories to store temporary data (keys) and serve cds binaries such as hatcheries and workers (download)
[directories]
download = "/app"
keys = "/app/keys"

###########################
# General server settings #
###########################
[server]
    [server.http]
    port = 8081
    sessionTTL = 60

    [server.grpc]
    port = 8082

    [server.secrets]
    key = ""
    # Uncomment this two lines to user a secret backend manager such as Vault. 
	# More details on https://github.com/ovh/cds/tree/configFile/contrib/secret-backends/secret-backend-vault
    # backend = "path/to/secret-backend-vault"
    # backendoptions = "vault_addr=https://vault.mydomain.net:8200 vault_token=09d1f099-3d41-666e-8337-492226789599 vault_namespace=/secret/cds"

################################
# Postgresql Database settings #
################################
[db]
user = "cds"
password = "cds"
name = "cds"
host = "localhost"
port = 5432
# DB SSL Mode: require, verify-full, or disable
sslmode = "disable"
maxconn = 20
timeout = 3000
# Uncomment this to retreive database credentials from secret-backend
# secret = "cds/db"
# The value must be as below
# {
#     "user": "STRING",
#     "password": "STRING"
# }

######################
# CDS Cache Settings #
######################
# If your CDS is made of a unique instance, a local cache if enough, but rememeber that all cached data will be lost on startup.
[cache]
#Uncomment to use redis as cache
#mode = "redis"
mode = "local"
ttl = 60
	# Connect CDS to a redis cache If you more than one CDS instance and to avoid losing data at startup
    [cache.redis]
    host = "localhost:6379" # If your want to use a redis-sentinel based cluster, follow this syntax ! <clustername>@sentinel1:26379,sentinel2:26379sentinel3:26379
    password = "your password"

##############################
# CDS Authentication Settings#
##############################
[auth]
# The default group is the group in which every new user will be granted at signup
defaultgroup = ""
# If Authentication is CDS local, you can switch between session based auth or basic auth
# localmode = "basic"
localmode = "session"

[auth.ldap]
enable = false
host = "<LDAP-server>"
port = 636
ssl = true
# LDAP Base
base = ""
# LDAP Bind DN
dn = "uid=%s,ou=people,{{.ldap-base}}"
# Define CDS user fullname from LDAP attribute
fullname = "{{.givenName}} {{.sn}}"

#####################
# CDS SMTP Settings #
#####################
[smtp]
disable = true
host = ""
port = 23
tls = false
user = ""
password = ""
from = "no-reply@cds.org"

##########################
# CDS Artifacts Settings #
##########################
# Either filesystem local storage or Openstack Swift Storage are supported
[artifact]
# mode = "swift#
mode = "local"

    [artifact.local]
    basedir = "/tmp/cds"

    [artifact.openstack]
    url = "<OS_AUTH_URL>"
    username = "<OS_USERNAME>"
    password = "<OS_PASSWORD>"
    tenant = "<OS_TENANT_NAME>"
    region = "<OS_REGION_NAME>"

#######################
# CDS Events Settings #
#######################
#For now, only Kafka is supported as a event broker
[events]
    [events.kafka]
    enabled = false
    broker = "<Kafka SASK/SSL addresses>"
    topic = "<Kafka topic>"
    user = "<Kafka username>"
    password = "<Kafka password>"

###########################
# CDS Schedulers Settings #
###########################
[schedulers]
disabled = false #This is mainly for dev purpose, you should not have to change it

####################
# CDS VCS Settings #
####################
[vcs]
    [vcs.polling]
    disabled = false #This is mainly for dev purpose, you should not have to change it

    [vcs.repositories]
    cacheloader_disabled = false #This is mainly for dev purpose, you should not have to change it

    [vcs.repositories.github]
    statuses_disabled = false # Set to true if you don't want CDS to push statuses on Github API
    statuses_url_disabled = false # Set to true if you don't want CDS to push CDS URL in statuses on Github API
    clientsecret = "" # You can define here your github client secret if you don't use secret-backend-manager

    [vcs.repositories.bitbucket]
    statuses_disabled = false 
    privatekey = "" # You can define here your bickcket private key if you don't use secret-backend-manager
```

## Environment Variable Configuration

```bash
CDS_URL_API
CDS_URL_UI
CDS_SERVER_HTTP_PORT
CDS_SERVER_HTTP_SESSIONTTL
CDS_SERVER_GRPC_PORT
CDS_SERVER_SECRETS_KEY
CDS_SERVER_SECRETS_BACKEND
CDS_SERVER_SECRETS_BACKEND_OPTION
CDS_LOG_LEVEL
CDS_LOG_DB
CDS_DB_USER
CDS_DB_PASSWORD
CDS_DB_NAME
CDS_DB_HOST
CDS_DB_PORT
CDS_DB_SSLMODE
CDS_DB_MAXCONN
CDS_DB_TIMEOUT
CDS_DB_SECRET
CDS_CACHE_MODE
CDS_CACHE_TTL
CDS_CACHE_REDIS_HOST
CDS_CACHE_REDIS_PASSWORD
CDS_DIRECTORIES_DOWNLOAD
CDS_DIRECTORIES_KEYS
CDS_AUTH_LOCALMODE
CDS_AUTH_LDAP_ENABLE
CDS_AUTH_LDAP_HOST
CDS_AUTH_LDAP_PORT
CDS_AUTH_LDAP_SSL
CDS_AUTH_LDAP_BASE
CDS_AUTH_LDAP_DN
CDS_AUTH_LDAP_FULLNAME
CDS_AUTH_DEFAULTGROUP
CDS_SMTP_DISABLE
CDS_SMTP_HOST
CDS_SMTP_PORT
CDS_SMTP_TLS
CDS_SMTP_USER
CDS_SMTP_PASSWORD
CDS_SMTP_FROM
CDS_ARTIFACT_MODE
CDS_ARTIFACT_LOCAL_BASEDIR
CDS_ARTIFACT_OPENSTACK_URL
CDS_ARTIFACT_OPENSTACK_USERNAME
CDS_ARTIFACT_OPENSTACK_PASSWORD
CDS_ARTIFACT_OPENSTACK_TENANT
CDS_ARTIFACT_OPENSTACK_REGION
CDS_EVENTS_KAFKA_ENABLED
CDS_EVENTS_KAFKA_BROKER
CDS_EVENTS_KAFKA_TOPIC
CDS_EVENTS_KAFKA_USER
CDS_EVENTS_KAFKA_PASSWORD
CDS_SCHEDULERS_DISABLED
CDS_VCS_POLLING_DISABLED
CDS_VCS_REPOSITORIES_CACHERLOADER_DISABLED
CDS_VCS_REPOSITORIES_GITHUB_STATUSES_DISABLED
CDS_VCS_REPOSITORIES_GITHUB_STATUSES_URL_DISABLED
CDS_VCS_REPOSITORIES_GITHUB_CLIENTSECRET
CDS_VCS_REPOSITORIES_BITBUCKET_STATUSES_DISABLED
CDS_VCS_REPOSITORIES_BITBUCKET_PRIVATEKEY

```