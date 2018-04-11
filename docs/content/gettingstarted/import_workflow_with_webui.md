+++
title = "Import a CDS Workflow from a repository"
weight = 2
+++

{{% notice note %}}

In this tutorial, you will create a CDS Workflow from an existing git repository.

* The repository have to add `.cds/` directory.
* With the web UI, on will create a CDS Workflow 'as code'.
{{% /notice %}}

## Prerequisites

 * Have an account on your CDS instance
 * Have a CDS project and a [Repository manager]({{< relref "/hosting/repositories-manager/_index.md" >}}) has been set up on your CDS Instance
 
## Prepare your git repository

The easiest way to initialize cds files in your repository is to follow [first tutorial]({{< relref "init_workflow_with_cdsctl.md" >}})

Create the pipeline file `.cds/build.pip.yml`

```yml
version: v1.0
name: build-pipeline
jobs:
- job: First job
  steps:
  - checkout: '{{.cds.workspace}}'
  - script : mvn package
  - artifactUpload : target/*.jar
  requirements:
  - binary: git
```

Create the application file `.cds/cdsdemo.app.yml`

```yml
version: v1.0
name: cdsdemo
vcs_server: github
repo: your-orga/cdsdemo
vcs_branch: '{{.git.branch}}'
vcs_default_branch: master
vcs_pgp_key: app-pgp-github

```

Create the workflow file `.cds/cdsdemo.yml`

```yml
name: cdsdemo
version: v1.0
pipeline: build-pipeline
payload:
  git.branch: "master"
  git.repository: yesnault/cdsdemo
application: cdsdemo  
pipeline_hooks:
- type: Git Repository Poller
```


## Create workflow from UI

* Attach a repository manager on your CDS Project - Advanced tab
![Attach repo manager](/images/getting_started_create_wf_ascode_ui_0_repo.png?height=400px)

* Go on Workflows tab, then click on 'Create Workflow'
![Workflow tab](/images/getting_started_create_wf_ascode_ui_1_wf_tab.png?height=400px)

* Click on 'From repository', then choose a repository manager
![Choose a repo manager - create workflow](/images/getting_started_create_wf_ascode_ui_2_from_repo.png?height=400px)

* Choose a git repository describe how to clone it, then click on 'Inspect repository'
![Choose repo](/images/getting_started_create_wf_ascode_ui_3_choose_repo.png?height=400px)

* Files found are display, then click on 'Create workflow' button
![Create workflow](/images/getting_started_create_wf_ascode_ui_4_create_wf.png?height=400px)

* A resume page is displayed, click on 'See workflow'
![Resume page](/images/getting_started_create_wf_ascode_ui_5_resume.png?height=400px)

* View workflow
![See Workflow](/images/getting_started_create_wf_ascode_ui_6_see_workflow.png?height=400px)