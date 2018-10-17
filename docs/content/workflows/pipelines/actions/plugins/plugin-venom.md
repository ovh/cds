+++
title = "plugin-venom"

+++

This plugin helps you to run venom. Venom: https://github.com/ovh/venom.

Add an extra step of type JUnit on your job to view tests results on CDS UI.


## Parameters

* **loglevel**: Log Level: debug, info, warn or error
* **vars**: Empty: all {{.cds...}} vars will be rewrited. Otherwise, you can limit rewrite to some variables. Example, enter cds.app.yourvar,cds.build.foo,myvar=foo to rewrite {{.cds.app.yourvar}}, {{.cds.build.foo}} and {{.foo}}. Default: Empty
* **vars-from-file**: filename.yaml or filename.json. See https://github.com/ovh/venom#run-venom-with-file-var
* **path**: Path containers yml venom files. Format: adirectory/, ./*aTest.yml, ./foo/b*/**/z*.yml
* **exclude**: Exclude some files, one file per line
* **output**: Directory where output XUnit result file
* **parallel**: Launch Test Suites in parallel. Enter here number of routines


## More

More documentation on [GitHub](https://github.com/ovh/cds/tree/master/contrib/grpcplugins/action/venom/README.md)


