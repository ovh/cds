+++
title = "plugin-venom"
chapter = true

[menu.main]
parent = "actions-plugins"
identifier = "plugin-venom"

+++

CDS plugin run venom https://github.com/runabove/venom

### Parameters

- **path** : Path containers yml venom files
- **exclude** : exclude some files, one file per line
- **parallel** : Launch Test Suites in parallel, default: 2
- **output** : Directory where output xunit result file

Add an extra step of type "junit" on your job to view results on CDS UI.

### More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-venom.md)
