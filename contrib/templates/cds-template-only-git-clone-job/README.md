# Plain Template

## Description

This template creates:
- a build pipeline with	one stage, containing one job
- job contains 2 steps: gitClone and a empty script.

Pipeline name contains Application name.
If you want to make a reusable pipeline, please consider updating this name after creation.

## Manual Build

```bash
cd $GOPATH/src/github.com/ovh/cds/contrib/templates/cds-template-only-git-clone-job
go build

# Create template on cds
cds templates add cds-template-only-git-clone-job

# Or Upload existing template on cds
cds templates update cds-template-only-git-clone-job cds-template-only-git-clone-job
``
