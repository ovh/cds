+++
title = "cds-nexus-upload"

[menu.main]
parent = "actions-user"
identifier = "cds-nexus-upload"

+++

Upload file on Nexus

## Parameters

* **files**: Regex of files you want to upload
* **repository**: Nexus repository that the artifact is contained in
* **artifactId**: Artifact id of the artifact
* **url**: Nexus URL
* **extension**: Extension of the artifact
* **groupId**: Group id of the artifact
* **version**: Version of the artifact. Supports resolving of 'LATEST', 'RELEASE' and snapshot versions ('1.0-SNAPSHOT') too.
* **packaging**: Packaging type of the artifact
* **login**: Login for nexus
* **password**: Password for nexus


## Requirements

* **bash**: type:binary Value:bash
* **curl**: type:binary Value:curl


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-nexus-upload.hcl)


