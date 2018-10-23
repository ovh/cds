+++
title = "cds-sonar-scanner"

+++

Run Sonar analysis. You must have a file sonar-project.properties in your source directory.

## Parameters

* **sonar-project.properties**: sonar-project.properties file
* **sonarBranch**: The Sonar branch (e.g. master)
* **sonarDownloadURL**: The download URL of Sonar CLI
* **sonarPassword**: The Sonar server's password
* **sonarURL**: The URL of the Sonar server
* **sonarUsername**: The Sonar server's username
* **sonarVersion**: SonarScanner's version to use
* **workspace**: The directory where your project is (e.g. /go/src/github.com/ovh/cds)


## Requirements

* **bash**: type: binary Value: bash
* **plugin-archive**: type: plugin Value: plugin-archive
* **plugin-download**: type: plugin Value: plugin-download


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-sonar-scanner.yml)


