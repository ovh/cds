# Description

*This is a demo tempate*

The Service Link prerequisite allow you to use any Docker image as a dependency of a job.

This is pretty useful if you want to make some tests with a real database, or put your builded application as a job prerequisite for doing some tests.

Doc: https://ovh.github.io/cds/docs/tutorials/service-requirement-pg/


# How to import it on your CDS Instance

This template is linked to group: `shared.infra`

If you want to import it, you have to be CDS Administrator on your CDS Instance.

This template uses a pre-requisite binary `apt-get`, you need a [worker model](https://ovh.github.io/cds/docs/concepts/worker-model/) with this capability on you CDS Instance.

``` bash
# import from github
cdsctl template push https://raw.githubusercontent.com/ovh/cds/master/contrib/workflow-templates/demo-usage-service-postgresql/demo-usage-service-postgresql.yml
```
