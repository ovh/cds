# Description

*This is a demo tempate*

The template will generates a skeleton of a workflow with usual steps:
- build and package
- deploy on development environment
- it test
- deploy on pre-production and production environments

# How to import it on your CDS Instance

This template is linked to group: `shared.infra`

If you want to import it, you have to be CDS Administrator on your CDS Instance.

``` bash
# import from github
cdsctl template push https://raw.githubusercontent.com/ovh/cds/master/contrib/workflow-templates/demo-workflow-skeleton/demo-workflow-skeleton.yml
```
