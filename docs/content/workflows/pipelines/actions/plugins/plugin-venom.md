+++
title = "plugin-venom"

[menu.main]
parent = "actions-plugins"
identifier = "plugin-venom"

+++

This plugin helps you to run venom. Venom: https://github.com/ovh/venom.

Add an extra step of type junit on your job to view tests results on CDS UI.

## Parameters

* **details**: Output Details Level: low, medium, high
* **exclude**: Exclude some files, one file per line
* **loglevel**: Log Level: debug, info, warn or error
* **output**: Directory where output xunit result file
* **parallel**: Launch Test Suites in parallel. Enter here number of routines
* **path**: Path containers yml venom files. Format: adirectory/, ./*aTest.yml, ./foo/b*/**/z*.yml
* **vars**: Empty: all {{.cds...}} vars will be rewrited. Otherwise, you can limit rewrite to some variables. Example, enter cds.app.yourvar,cds.build.foo,myvar=foo to rewrite {{.cds.app.yourvar}}, {{.cds.build.foo}} and {{.foo}}. Default: Empty
* **vars-from-file**: filename.yaml or filename.json. See https://github.com/ovh/venom#run-venom-with-file-var


## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-venom/README.md)

