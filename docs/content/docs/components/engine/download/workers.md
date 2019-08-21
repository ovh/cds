---
title: "workers"
notitle: true
notoc: true
---
# engine download workers

`Download workers binaries from latest release on Github`

## Synopsis

Download workers binaries from latest release on Github

You can also indicate a specific os or architecture to not download all binaries available with flag --os and --arch

```
engine download workers [flags]
```

## Examples

```
engine download workers
```

## Options

```
      --arch string                Download only for this arch
      --config string              config file
      --os string                  Download only for this os
      --remote-config string       (optional) consul configuration store
      --remote-config-key string   (optional) consul configuration store key (default "cds/config.api.toml")
```

## SEE ALSO

* [engine download](/docs/components/engine/download/)	 - `Download binaries`

