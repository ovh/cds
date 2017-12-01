# CDS Template for building template

## Description

This template creates a pipeline for building CDS Template.

Template contains:
- A "Commit Stage" with one job "Compile"
- Job contains two steps: GitClone and CDS_GoBuild

Once builded, you can download and import your template with CDS CLI:

```bash
cds admin templates add your-template
```

## Manual Build

```bash
cd $GOPATH/src/github.com/ovh/cds/contrib/templates/cds-template-cds-template
go build

# Create template on cds
cds templates add cds-template-cds-template

# Or Upload existing template on cds
cds templates update cds-template-cds-template cds-template-cds-template
``
