# CDS Plugin Template

## Description

This template creates a pipeline for building CDS Plugin.

Template contains:
- A "Commit Stage" with one job "Compile"
- Job contains two steps: CDS_GitClone and CDS_GoBuild

## Manual Build

```bash
cd $GOPATH/src/github.com/ovh/cds/contrib/templates/template-cds-plugin
go build

# Create template on cds
cds templates add template-cds-plugin

# Or Upload existing template on cds
cds templates update template-cds-plugin template-cds-plugin
``
