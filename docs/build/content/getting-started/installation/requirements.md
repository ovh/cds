+++
title = "Requirements"

[menu.main]
parent = "installation"
identifier = "installation-requirements"
weight = 1

+++


CDS API is the core component of CDS.
To start CDS api, the **only mandatory dependency is a PostgreSQL database** and a path to the directory containing other CDS binaries.

There is are two ways to set up CDS:

- as [toml](https://github.com/toml-lang/toml) configuration
- over environment variables.

## CDS API Third-parties

At the minimum, CDS needs a PostgreSQL Database >= 9.4. But for serious usage your may need :

- A [Redis](https://redis.io) server or sentinels based cluster used as a cache and session store
- A LDAP Server for authentication
- A SMTP Server for mails
- A [Kafka](https://kafka.apache.org/) Broker to manage CDS events
- A [Openstack Swift](https://docs.openstack.org/developer/swift/) Tenant to store builds artifacts
- A [Vault](https://www.vaultproject.io/) server for cipher and app keys
- A [Consul](https://www.consul.io/) to manage CDS Configuration

See Configuration template for more details


## Supported Platforms
