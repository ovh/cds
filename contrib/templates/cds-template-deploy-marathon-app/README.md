# Template deploy marathon application

## Description

This template creates a deploy pipeline with one stage: Deploy Stage.

The stage calls plugin-marathon to deploy an application on marathon.

An application variable ```marathon.config``` contains the marathon configuration content file.

## Manual Build

```bash
cd $GOPATH/src/github.com/ovh/cds/contrib/templates/template-plain
go build

# Create template on cds
cds templates add template-plain

# Or Upload existing template on cds
cds templates update template-plain template-plain
``
