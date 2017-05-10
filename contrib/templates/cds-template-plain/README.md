# Plain Template

## Description

This sample template creates:
- a build pipeline with	two stages: Commit Stage and Packaging Stage
- a deploy pipeline with one stage: Deploy Stage

Commit Stage:
- run git clone
- run make build

Packaging Stage:
- run docker build and docker push

Deploy Stage:
- it's an empty script

Packaging and Deploy are optional.

Compile and deploy it.

## Manual Build

```bash
cd $GOPATH/src/github.com/ovh/cds/contrib/templates/template-plain
go build

# Create template on cds
cds templates add template-plain

# Or Upload existing template on cds
cds templates update template-plain template-plain
``
