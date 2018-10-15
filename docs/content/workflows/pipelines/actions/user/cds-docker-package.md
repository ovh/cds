+++
title = "cds-docker-package"

+++

Build image and push it to Docker repository

## Parameters

* **dockerOpts**: Docker options, Enter --no-cache --pull if you want for example
* **dockerRegistry**: Docker Registry. Enter myregistry for build image myregistry/myimage:mytag
* **dockerRegistryPassword**: Docker Registry Password. Enter password to connect on your Docker registry.
* **dockerRegistryUsername**: Docker Registry Username. Enter username to connect on your Docker registry.
* **dockerfileDirectory**: Directory which contains your Dockerfile.
* **imageName**: Name of your Docker image, without tag. Enter myimage for build image myregistry/myimage:mytag
* **imageTag**: Tag og your Docker image.
Enter mytag for build image myregistry/myimage:mytag. {{.cds.version}} is a good tag from CDS.
You can use many tags: firstTag,SecondTag
Example: {{.cds.version}},latest


## Requirements

* **docker**: type: binary Value: docker


More documentation on [GitHub](https://github.com/ovh/cds/tree/master/contrib/actions/cds-docker-package.yml)


