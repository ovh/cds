# Description

*This is a demo tempate*

The template will generates a Hello World workflow with usual steps:
- build and package
- deploy on development environment
- it test
- deploy on pre-production and production environments

It's the same as demo-workflow-skeleton workflow template, but does not contain dependency with a Git Repository Manager. So that, it's pretty
useful to discover CDS Workflow with the [Ready To Run](https://ovh.github.io/cds/hosting/ready-to-run/docker-compose/) without having a Git
repository manager attached.

# How to import it on your CDS Instance

This template is linked to group: `shared.infra`

If you want to import it, you have to be CDS Administrator on your CDS Instance.

``` bash
# import from github
cdsctl template push https://raw.githubusercontent.com/ovh/cds/master/contrib/workflow-templates/demo-workflow-hello-world/demo-workflow-hello-world.yml
```
