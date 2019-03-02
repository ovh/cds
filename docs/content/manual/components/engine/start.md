+++
title = "start"
+++
## engine start

`Start CDS`

### Synopsis


Start CDS Engine Services

#### API

This is the core component of CDS.


#### Hatcheries

They are the components responsible for spawning workers. Supported integrations/orchestrators are:

* Local machine
* Openstack
* Docker Swarm
* Openstack
* Vsphere

#### Hooks
This component operates CDS workflow hooks

#### Repositories
This component operates CDS workflow repositories

#### VCS
This component operates CDS VCS connectivity

Start all of this with a single command:

	$ engine start [api] [hatchery:local] [hatchery:marathon] [hatchery:openstack] [hatchery:swarm] [hatchery:vsphere] [elasticsearch] [hooks] [vcs] [repositories] [migrate]

All the services are using the same configuration file format.

You have to specify where the toml configuration is. It can be a local file, provided by consul or vault.

You can also use or override toml file with environment variable.

See $ engine config command for more details.



```
engine start [flags]
```

### Options

```
      --config string              config file
  -h, --help                       help for start
      --remote-config string       (optional) consul configuration store
      --remote-config-key string   (optional) consul configuration store key (default "cds/config.api.toml")
      --vault-addr string          (optional) Vault address to fetch secrets from vault (example: https://vault.mydomain.net:8200)
      --vault-token string         (optional) Vault token to fetch secrets from vault
```

### SEE ALSO

* [engine](/manual/components/engine/engine/)	 - CDS Engine

