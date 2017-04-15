# CDS Plugin Template

## Description

This template creates a pipeline for building CDS Plugin.

Template contains:
- A "Commit Stage" with one job "Compile"
- Job contains two steps: GitClone and CDS_GoBuild

Once builded, you can download then import your plugin with CDS CLI:

```bash
cds admin plugins add your-plugin
```

## Manual Build

```bash
cd $GOPATH/src/github.com/ovh/cds/contrib/templates/cds-template-cds-plugin
go build

# Create template on cds
cds templates add cds-template-cds-plugin

# Or Upload existing template on cds
cds templates update cds-template-cds-plugin cds-template-cds-plugin
``
