+++
title = "cds-docker-package"

[menu.main]
parent = "actions-user"
identifier = "cds-docker-package"

+++

Build image and push it to docker repository

## Parameters

* **dockerOpts**: Docker options, Enter --no-cache --pull if you want for example
* **dockerRegistry**: Docker Registry. Enter myregistry for build image myregistry/myimage:mytag
* **dockerfileDirectory**: Directory which contains your Dockerfile.
* **imageName**: Name of your docker image, without tag. Enter myimage for build image myregistry/myimage:mytag
* **imageTag**: Tag of your docker image.
Enter mytag for build image myregistry/myimage:mytag. {{.cds.version}} is a good tag from CDS.
You can use many tags: firstTag,SecondTag
Example : {{.cds.version}},latest


## Requirements

* **bash**: type: binary Value: bash
* **docker**: type: binary Value: docker


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-docker-package.hcl)


