+++
title = "cds-template-cds-plugin"
chapter = true

[menu.main]
parent = "templates"
identifier = "cds-template-cds-plugin"

+++


This template creates a pipeline for building CDS Plugin with:

- A "Commit Stage" with one job "Compile"
- Job contains two steps: GitClone and CDS_GoBuild


## Parameters

* **repo**: Your source code repository
* **package.root**: example: github.com/ovh/cds
* **package.sub**: Directory inside your repository where is the plugin.
Enter "contrib/plugins/your-plugin" for github.com/ovh/cds/contrib/plugins/your-plugin
			


## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/templates/cds-template-cds-plugin/README.md)

