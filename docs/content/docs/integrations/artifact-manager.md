---
title: Artifactory
main_menu: true
card: 
  name: artifact-manager
---

The Artifactory integration uses the builtin integration model "Artifact Manager" and can be configured on every project by users

This integration allows you:

* to upload/download artifact into artifactory
* to create a new artifactory build from your workflow run results
* to create a release bundle using artifacts previously uploaded in artifactory

## Recommendations

To take full advantage of this integration, we recommend a few things:

* Naming your local repositories like this: [team]-[technology]-[maturity]
* Having virtual repositories named like this: [team]-[technology]

For example, you need a generic repository for CDS artifacts. You can create something like:

* Virtual repository: myteam-docker
* Snapshot repository: myteam-docker-snapshot
* Release repository: myteam-docker-release

By using this convention, the CDS release action will be able to promote your artifacts from snapshot to release repositories

## How to configure Artifactory integration on your project

### Link Artifactory to your project

On the integration project view, add a new "Artifact Manager" integration and fill the following parameters:

* `name`: The name of the integration.
* `platform`: Must be 'artifactory'
* `url`: URL of artifactory api (https//myinstance.ofartifactory/artifactory/)
* `project.key`: The name of the artifactory project (https://www.jfrog.com/confluence/display/JFROG/Projects)
* `cds.repository`: The name of the repository used by CDS to upload/download artifacts (must be a virtual repository)
* `token.name`: The name of the access token used by CDS to access the artifactory API
* `token`: The value of the access token used by CDS to access the artifactory API
* `release.token`: The value of the access token used by CDS to access the distribution API (https://www.jfrog.com/confluence/display/JFROG/JFrog+Distribution)
* `promotion.maturity.low`: suffix used on your local repositories to identify your snapshots

### Enable Artifactory integration on your workflow

On the workflow advanced view, you can link your workflow to project integration.

## Integration actions

The artifactory integration comes with 5 actions (https://github.com/ovh/cds/tree/master/contrib/integrations/artifactory)

### Artifactory-Upload-Artifact

This plugin is used by CDS Upload Artifact action to send artifact into artifactory.

The artifacts will be stored in the cds repository provided during the integration configuration (cds.repository)

### Artifactory-Download-Artifact

This plugin is used by CDS Download Artifact action to retrieve artifact from artifactory. 

The artifacts will be downloaded from the cds repository provided during the integration configuration (cds.repository)

### Artifactory-Push-Build-Info

This plugin is used by CDS Push Build Info action to create inside artifactory a build-info (https://www.jfrog.com/confluence/display/JFROG/Build+Integration).

This action must be run after all the artifacts have been uploaded

The build name computed by CDS will be: [build.info.path]/[cds.projectkey]/[cds.workflow.name]

### Artifactory-Promote

This plugin is used by CDS Promote action to move artifacts from 1 repository to another. The two repositories must share the name but having a different suffix.

For example:
 * my-docker-repo-snapshot
 * my-docker-repo-release


### Artifactory-Release

This plugin is used by CDS Release action. 

It promotes the provided artifacts, create a release bundle and distributes it on all the edges.

This action use both the artifactory and distribution APIs.
