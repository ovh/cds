# CDS: Continuous Delivery Service

<img align="right" src="https://raw.githubusercontent.com/ovh/cds/master/logo-background.png">

CDS is a pipeline based Continuous Delivery μservice written in Go.

**/!\ This project is under active development.**

## Documentation

Documentation is available [here](/doc/overview/introduction.md)

## Overview

CDS is a μservice composed of 4 different Go binaries:

 * api
 * [worker](/doc/overview/worker.md)
 * [hatchery](/doc/overview/hatchery.md)
 * cli

An WebUI is also available at https://github.com/ovh/cds-ui

### SDK

A Go SDK is available at github.com/ovh/cds/sdk. It provide helper functions for all API handlers, with embedded authentification mechanism.

### CDS Contrib

Actions, Plugins, Templates, uServices are under : https://github.com/ovh/cds/tree/master/contrib

### API usage

To start CDS api, the only mandatory dependency is a PostgreSQL database dsn and a path to the directory containing other CDS binaries. Ex:

```
$ ./api --db-host=127.0.0.1 --db-user=cds --db-password=XX --download-directory=$GOPATH/bin
```

To get the best out of CDS api though, one should use all compatible third parties to ensure maximum security and availability:

 - Openstack Swift for artifact storage
 - Vault for cipher and app keys
 - SMTP for mail notification
 - SSL for end-to-end encrypted communication
 - Redis for caching
 - LDAP for user management


## Advanced API usage

### Vault

It is possible to configure CDS to fetch secret cipher keys from Vault.

See documentation at https://github.com/ovh/cds/tree/master/contrib/secret-backends/secret-backend-vault

### Artifact Storage

 Artifacts are either stored on API filesystem or on Openstack Swift to garantee High Availabilty.

```
 --artifact-mode string                Artifact Mode: openstack or filesystem (default "filesystem")
```

### Caching

 Cache from database is enabled in process by default. To avoid high memory consumption, Redis caching is available.

```
 --cache string                        Cache : local|redis (default "local")
 --cache-ttl int                       Cache Time to Live (seconds) (default 600)
 --redis-host string                   Redis hostname (default "localhost:6379")
 --redis-password string               Redis password
```

### Notification

### SMTP

SMTP should be enabled to allow user account creation.

```
 --smtp-from string                    SMTP From
 --smtp-host string                    SMTP Host
 --smtp-password string                SMTP Password
 --smtp-port string                    SMTP Port
 --smtp-tls                            SMTP TLS
 --smtp-user string                    SMTP Username
```


### LDAP

Users can be fetched from an external LDAP. If activated, user creation directly in CDS is disabled.

```
 --ldap-base string                    LDAP Base
 --ldap-dn string                      LDAP Bind DN (default "uid=%s,ou=people,{{.ldap-base}}")
 --ldap-enable                         Enable LDAP Auth mode : true|false
 --ldap-host string                    LDAP Host
 --ldap-port int                       LDAP Post (default 636)
 --ldap-ssl                            LDAP SSL mode (default true)
 --ldap-user-fullname string           LDAP User fullname (default "{{.givenName}} {{.sn}}")
```

### Database

```
 --db-host string                      DB Host (default "localhost")
 --db-maxconn int                      DB Max connection (default 20)
 --db-name string                      DB Name (default "cds")
 --db-password string                  DB Password
 --db-port string                      DB Port (default "5432")
 --db-sslmode string                   DB SSL Mode: require (default), verify-full, or disable (default "require")
 --db-timeout int                      Statement timeout value (default 3000)
 --db-user string                      DB User (default "cds")
```

### Logging

API logs can either be printed on stdout or send in a dedicated table in database

```
 --db-logging                          Logging in Database: true of false
```

## Database management

CDS provide all needed tools scripts to perform Schema creation and auto-migration. Those tools are embedded inside the `api` binary.

### Creation

On a brand new database run the following command:

```shell
    $ cd $GOPATH/src/github/ovh/cds
    $ engine/api/api database upgrade --db-host <host> --db-host <port> --db-password <password> --db-name <database> --limit 0
```

### Upgrade

On an existing database, run the following command on each CDS update:

```shell
    $ cd $GOPATH/src/github/ovh/cds
    $ engine/api/api database upgrade --db-host <host> --db-host <port> --db-password <password> --db-name <database>
```

### More details

[Read more about CDS Database Management](https://github.com/ovh/cds/tree/master/engine/sql)


## Links

- *OVH home (us)*: https://www.ovh.com/us/
- *OVH home (fr)*: https://www.ovh.com/fr/
- *OVH community*: https://community.ovh.com/c/open-source/continuous-delivery-service


## License

3-clause BSD
