+++
title = "plugin-group-tmpl"

[menu.main]
parent = "actions-plugins"
identifier = "plugin-group-tmpl"

+++

This actions helps you generate a marathon group application file.
It takes a config template file as a single application, and creates the group with the variables specified for each application in the applications files.
Check documentation on text/template for more information https://golang.org/pkg/text/template.

## Parameters

* **applications**: Applications file variables
* **config**: Template file to use
* **output**: Output path for generated file (default to <file>.out or just trimming .tpl extension)


## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-group-tmpl/README.md)

