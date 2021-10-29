---
title: "Requirements"
weight: 2
card: 
  name: operate
---


CDS API is the core component of CDS.
To start CDS API, the **only mandatory dependencies are a PostgreSQL database, a Redis server** and a path to the directory containing other CDS binaries.

There is are two ways to set up CDS:

- as [toml](https://github.com/toml-lang/toml) configuration
- over environment variables.

## CDS API Third-parties

At the minimum, CDS needs a PostgreSQL database >= 9.6 (version 14.0 recommanded) and Redis >= 3.2. But for serious usage your may need:

- A [Redis](https://redis.io) server or sentinels based cluster used as a cache and session store
- A LDAP Server for authentication
- A SMTP Server for mails
- A [Kafka](https://kafka.apache.org/) Broker to manage CDS events
- A [OpenStack Swift](https://docs.openstack.org/developer/swift/) Tenant to store builds artifacts
- A [Vault](https://www.vaultproject.io/) server for CDS configuration
- A [Consul](https://www.consul.io/) to manage CDS Configuration

See Configuration template for more details


## Supported Platforms

- Darwin i386 / amd64
- FreeBSD amd64
- Linux i386 / amd64 / arm(Raspberry Pi) / arm64 / ppc64le (Power 8 & 9)
- OpenBSD amd64
- Solaris amd64
- Windows amd64

You'll find binaries on [CDS Releases](https://github.com/ovh/cds/releases/latest)
