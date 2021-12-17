---
title: "CDN"
weight: 3
---

## What's CDN

CDN is a service dedicated to receive and store CDS's logs and artifacts. 

CDN stores the list of all known log or artifact items in a Postgres database and communicates with storage backends to store the contents of those items.
These backends are call units and there are two types of units in CDN:

* Buffer unit: To store logs and artifacts of incoming jobs, these units are designed to be fast for read / write operations, but with limited capacity.

* Storage Unit: to store complete job's logs and artifacts.

When logs or file are received by CDN from a cds worker, it will first store these items in its buffer. Then, when the item is fully received, it will be moved to one of the configured storage units.
If the CDN service is configured with multiple storage units, each unit periodically checks for missing items and synchronizes these items from other units.

CDS UI and CLI communicate with CDN to get entire logs, or stream them.

## Supported units
* Buffer (type: log): Redis.
* Buffer (type: file): Local, NFS
* Storage: Local, Swift, S3, Webdav

## Configuration
Like any other CDS service, CDN requires to be authenticated with a consumer. The required scopes are `Service`, `Worker` and `RunExecution`.

You can generate a configuration file with the `engine` binary:

```sh
$ engine config new cdn > cds-configuration.toml
```

You must have at least one storage unit, one file buffer and one log buffer to be able to run CDN.

### CDN artifact configuration

#### Storage Unit Buffer

You must have a `storageUnits.buffers` , one for the type `log`, another for the type `file`.

Type `log`:

```toml
      [cdn.storageUnits.buffers.redis]
        bufferType = "log"

        [cdn.storageUnits.buffers.redis.redis]
          host = "aaa@instance0,instance1,instance2"
          password = "your-password"
```

Type `file`:

```toml
      [cdn.storageUnits.buffers.local-buffer]

        # it can be 'log' to receive logs or 'file' to receive artifacts
        bufferType = "file"

        [cdn.storageUnits.buffers.local-buffer.local]
          path = "/var/lib/cds-engine/cdn-buffer"
```

To multi-instanciate the cdn service, you can use a NFS for the bufferType file, example:
    
```toml
      [cdn.storageUnits.buffers.buffer-nfs]
        bufferType = "file"
 
        [cdn.storageUnits.buffers.buffer-nfs.nfs]
          host = "w.x.y.z"
          targetPartition = "/zpool-partition/cdn"
          userID = 0
          groupID = 0
 
          [[cdn.storageUnits.buffers.buffer-nfs.nfs.encryption]]
            Cipher = "aes-gcm"
            Identifier = "nfs-buffer-id"
            ## enter a key here, 32 lenght
            Key = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
            Sealed = false   
```

#### Storage Units Storage

The storage unit 'storage' store the artifacts. 
You can use `Local`, `Swift`, `S3`, `Webdav`

Example of storage unit `local`:

```toml
    [cdn.storageUnits.storages]

      [cdn.storageUnits.storages.local]

        # flag to disabled backend synchronization
        disableSync = false

        # global bandwith shared by the sync processes (in Mb)
        syncBandwidth = 128

        # number of parallel sync processes
        syncParallel = 2

        [cdn.storageUnits.storages.local.local]
          path = "/tmp/cds/local-storage"

          [[cdn.storageUnits.storages.local.local.encryption]]
            Cipher = "aes-gcm"
            Identifier = "cdn-storage-local"
            LocatorSalt = "xxxxxxxxx"
            SecretValue = "xxxxxxxxxxxxxxxxx"
            Timestamp = 0
```

Example of storage unit `swift`:
```
    [cdn.storageUnits.storages]

      [cdn.storageUnits.storages.swift]
        syncParallel = 6
        syncBandwidth = 1000

        [cdn.storageUnits.storages.swift.swift]
          address = "https://xxx.yyy.zzz/v3"
          username = "foo"
          password = "your-password-here"
          tenant = "your-tenant-here"
          domain = "Default"
          region = "XXX"
          containerPrefix = "prod"

          [[cdn.storageUnits.storages.swift.swift.encryption]]
            Cipher = "aes-gcm"
            Identifier = "swift-backend-id"
            LocatorSalt = "XXXXXXXX"
            SecretValue = "XXXXXXXXXXXXXXXX"
```