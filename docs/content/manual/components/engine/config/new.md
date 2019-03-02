+++
title = "new"
+++
## engine config new

`CDS configuration file assistant`

### Synopsis


# generate the whole configuration file
	$ engine config new > conf.toml

# you can compose your file configuration
# this will generate a file configuration containing
# api and hatchery:local µService
	$ engine config new api hatchery:local

# For advanced usage, Debug and Tracing section can be generated as:
	$ engine config new debug tracing [µService(s)...]

# All options
	$ engine config new [debug] [tracing] [api] [hatchery:local] [hatchery:marathon] [hatchery:openstack] [hatchery:swarm] [hatchery:vsphere] [elasticsearch] [hooks] [vcs] [repositories] [migrate]



```
engine config new [flags]
```

### Options

```
      --env    Print configuration as environment variable
  -h, --help   help for new
```

### SEE ALSO

* [engine config](/manual/components/engine/config/)	 - `Manage CDS Configuration`

