+++
title = "Write a Plugin"
weight = 10

[menu.main]
parent = "plugins"
identifier = "plugins-write_plugin"

+++

A CDS worker executes job, and job is composed of steps.

A step is :

* a builtin action, as GitClone, etc... [read more]({{< relref "workflows/pipelines/actions/builtin/_index.md" >}})
* a user action [read more]({{< relref "workflows/pipelines/actions/user-actions/_index.md" >}})
* a Plugin Action

A Plugin is a Golang Binary.

Take a look at https://github.com/ovh/cds/tree/master/sdk/plugin/dummy/dummy_plugin.go

Contribute on https://github.com/ovh/cds/tree/master/contrib/plugins
