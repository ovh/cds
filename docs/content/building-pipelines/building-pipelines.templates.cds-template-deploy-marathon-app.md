+++
title = "cds-template-deploy-marathon-app"
chapter = true

[menu.main]
parent = "templates"
identifier = "cds-template-deploy-marathon-app"

+++


This template creates:

- a deployment pipeline with one stage, and containing one job
- job calls plugin-marathon
- an application with a variable named "marathon.config"
- uses environment variables marathonHost, password and user

Please update Application / Environment Variables after creating application.


## Parameters

* **docker.image**: Your docker image without the tag
* **marathon.appID**: Your marathon application ID
* **marathon.config**: Content of your marathon.json file


## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/templates/cds-template-deploy-marathon-app/README.md)

