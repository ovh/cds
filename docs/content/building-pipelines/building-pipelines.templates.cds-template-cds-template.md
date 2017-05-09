+++
title = "cds-template-cds-template"
chapter = true

[menu.main]
parent = "templates"
identifier = "cds-template-cds-template"

+++


This template creates a pipeline for building CDS Template with:

- A "Commit Stage" with one job "Compile"
- Job contains two steps: GitClone and CDS_GoBuild


## Parameters

* **package.root**: example: github.com/ovh/cds
* **package.sub**: Directory inside your repository where is the template.
Enter "contrib/templates/your-plugin" for github.com/ovh/cds/contrib/templates/your-plugin
			
* **repo**: Your source code repository


## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/templates/cds-template-cds-template/README.md)

