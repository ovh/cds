+++
title = "Configuration"
weight = 3

[menu.main]
parent = "installation"
identifier = "configuration"

+++

The toml configuration can be provided by a file, via [consul k/v store](https://www.consul.io) or via [vault](https://www.vaultproject.io/).

### Start CDS with local configuration file

You can also generate a configuration file template with the following command.

```bash
$ $PATH_TO_CDS/engine config new > my_conf_file.toml
```

Edit this file.

Check your configuration file with

```bash
$ $PATH_TO_CDS/engine config check my_conf_file.toml
Reading configuration file my_new_file.toml
Configuration file OK
```

Create your database relations

```bash
$ $PATH_TO_CDS/engine database upgrade --db-host <host> --db-port <port> --db-user <user> --db-password <password> --db-name <database> --migrate-dir $PATH_TO_CDS/engine/sql --limit 0
```

Run CDS

```bash
$ $PATH_TO_CDS/engine start api --config my_conf_file.toml
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
$ $PATH_TO_CDS/engine start api --remote-config localhost:8500 --remote-config-key cds/config.api.toml
Reading configuration from localhost:8500
2017/04/04 16:11:25 [NOTICE]   Starting CDS server...
...
```

### Start CDS with Vault

You have to put your configuration in a TOML format like above with good values into a secret named `/secret/cds/conf` in your vault.
For example if you use the vault CLI :

```bash
$ myConfig=`cat conf.toml`
$ vault write secret/cds/conf data=$myConfig
```

```bash
$ $PATH_TO_CDS/engine start api --vault-addr=http://myvault.com  --vault-token=XXXX
Reading configuration from vault @http://myvault.com
2017/04/04 16:33:17 [NOTICE]   Starting CDS server...
```
