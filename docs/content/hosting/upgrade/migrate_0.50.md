---
title: "Migrate 0.50"
weight: 1
---

## Migrate an existing instance

Before upgrading your CDS Instance:
- You have to backup your databases: cds and cdn databases.
- You have to install the version 0.49.0 if you use the 0.48 version.
- The backend cds-backend have to be removed from the cdn configuration.

## PostgreSQL

PostgreSQL 14 is now recommanded

## CDN Service

This release introduced the CDN Artifacts. This means that all artifacts upload / download will be done through the CDN service.
This is not enabled by default, you have to enable that with a feature flipping. The version 0.51 will use the CDN service to manage artifacts by default.

### How to configure the new CDN artifact service?

Some configuration have to be added to the cdn service to manage artifacts.

You have to add a `storageUnits.buffers` with the type "file".
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

## How to enable the CDS Artifact?

Make sure you have configured cdn in the paragraph above, then enable it with this command:

```sh
cat <<EOF > feature.yaml
name: cdn-artifact
rule: return true
EOF
cdsctl admin feature import feature.yaml
```

In the next release (0.51.0), you don't need to use the feature flipping `cdn-artifact` to enable cdn artifact.

### How to migrate existing artifacts to the new backend?

If you want to migrate some artifacts to the new backend, you can use cdsctl:
```sh
$ cdsctl workflow artifact cdn-migrate <project_key> <workflow_name> <run_number>
```
This command will migrate artifacts for one workflow run.

## Workflow Runs : Purge and Retention

The default retention rule have to be added in api configuration:

```toml
  [api.workflow]

    # Default rule for workflow run retention policy, this rule can be overridden on each workflow.
    # Example: 'return run_days_before < 365' keeps runs for one year.
    defaultRetentionPolicy = "return run_days_before < 365"
```
Documentation: https://ovh.github.io/cds/docs/concepts/workflow/retention/

## Spawning worker : MaxAttemptsNumberBeforeFailure

A new hatchery configuration attribute is available to control the maximum attempts to start a same job.

Example on the `hatchery.local`:

```toml
      [hatchery.local.commonConfiguration.provision]

        # Maximum attempts to start a same job. -1 to disable failing jobs when too many attempts
        # maxAttemptsNumberBeforeFailure = 5
```

## CDS Binaries lazy loading

`downloadFromGitHub` and `supportedOSArch` are added. This will allow you to download cds workers / cdsctl from GitHub if it's not in already downloaded into the `directory`.

```toml
  [api.download]

    # this directory contains cds binaries. If it's empty, cds will download binaries from GitHub (property downloadFromGitHub) or from an artifactory instance (property artifactory) to it
    directory = "/var/lib/cds-engine"

    # allow downloading binaries from GitHub
    downloadFromGitHub = true

    # example: ["darwin/amd64","darwin/arm64","linux/amd64","windows/amd64"]. If empty, all os / arch are supported: windows,darwin,linux,freebsd,openbsd and amd64,arm,386,arm64,ppc64le
    supportedOSArch = []
```

