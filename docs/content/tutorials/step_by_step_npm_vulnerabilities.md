+++
title = "Check npm vulnerabilities"
weight = 1
+++

{{% notice note %}}

In this tutorial, you will create a CDS Workflow with the Web UI that check javascript vulnerabilities

* Create a workflow using one pipeline
* You will discover the [Npm-audit-parser]({{< relref "plugin-npm-audit-parser.md" >}}) plugin action, which parse npm audit report

{{% /notice %}}

### 1 - Create your CDS project

Let's create a project. On the top navbar, click on 'Create a project'.

* Enter a Project Name
* The project key will be useful when you want to use [cdsctl]({{< relref "cli/cdsctl/_index.md" >}}).
* Click on 'Create' button.

{{%expand "view screenshots..." %}}

![create_prj](/images/tutorials/npm-audit-parser/create_prj.png?height=400px&classes=shadow)

{{% /expand%}}

### 2 - Add a repository manager on your project

The project is now create, you have to link a repository manager. 
Be sure to have a [Repository manager]({{< relref "/hosting/repositories-manager/_index.md" >}}) setted up on your CDS Instance.

* Select 'Advanced' section
* In the section 'Link to a repository manager', select 'github', then click on 'Connect'
* A popup is displayed, Click on 'Click here' to finalize the link with github. By doing that, you allow CDS to create hook on github.


{{%expand "view screenshots..." %}}

Select 'Advanced' section
![admin_prj](/images/tutorials/npm-audit-parser/admin_prj.png?height=400px&classes=shadow)

In the section 'Link to a repository manager', select 'github', then click on 'Connect'
![admin_prj_add_repo](/images/tutorials/npm-audit-parser/admin_prj_add_repo.png?height=400px&classes=shadow)

A popup is displayed, Click on 'Click here' to finalize the link with github. By doing that, you allow CDS to create hook on github.
![admin_prj_add_repo_popup](/images/tutorials/npm-audit-parser/admin_prj_add_repo_popup.png?height=400px&classes=shadow)

Here, Github is well linked on your CDS project.

![admin_repo_linked](/images/tutorials/npm-audit-parser/admin_repo_linked.png?height=400px&classes=shadow)

{{% /expand%}}

### 3 - Create an application, link it to a Git Repository

You've got a project, linked to Github. Let's create an application.

A CDS Application is useful to have a link to a Git Repository.

* Go on Project -> Applications tab, click on 'Create a new application'
* Enter application name, here 'my-node-app'
* Go on Advanced tab, select a Repository
* Expand configuration, select 'https'. If your repository is public, you can keep empty fields, then click on 'Save'

{{%expand "view screenshots..." %}}

Go on Project -> Applications tab, click on 'Create a new application'
![app_tab](/images/tutorials/npm-audit-parser/app_tab.png?height=400px&classes=shadow)

Enter application name, here 'my-node-app'
![create_app](/images/tutorials/npm-audit-parser/create_app.png?height=400px&classes=shadow)

Go on Advanced tab, select a Repository
![admin_app](/images/tutorials/npm-audit-parser/admin_app.png?height=400px&classes=shadow)

Expand configuration, select 'https'. If your repository is public, you can keep empty fields, then click on 'Save'
![app_link_repo](/images/tutorials/npm-audit-parser/app_link_repo.png?height=400px&classes=shadow)

{{% /expand%}}


### 4 - Create the workflow

* Go to Project -> Worflows tab
* Enter the Workflow name, then click on Next
* You have now to choose the first pipeline. As you don't have a pipeline yet, you will create a new one, named 'check-node-vunerabilities'. Click on Next
* Now, you have to select an application. Choose your application 'my-node-app', then click on Next
* We don't need an environment, neither plaform for the build pipeline, Click on 'Finish'
* Edit the pipeline 'check-node-vunerabilities'
* Click on Edit as Code button, then paste that code:

```yml
version: v1.0
name: check-node-vunerabilities
jobs:
- job: New Job
  steps:
  - checkout: '{{.cds.workspace}}'
  - script:
    - npm install --no-audit
  - optional: true
    script:
    - npm audit --json > report.json
  - plugin-npm-audit-parser:
      file: report.json
  requirements:
  - binary: git
  - binary: npm
  - plugin: plugin-npm-audit-parser
```

{{%expand "view screenshots..." %}}

Create the pipeline
![create_pipeline](/images/tutorials/npm-audit-parser/create_pipeline.png?height=400px&classes=shadow)

Then select your application.
![select_app](/images/tutorials/npm-audit-parser/select_app.png?height=400px&classes=shadow)

Click on create to create the workflow.
![create](/images/tutorials/npm-audit-parser/create_wf.png?height=400px&classes=shadow)

Click on 'Edit the pipeline'
![click_edit](/images/tutorials/npm-audit-parser/click_edit.png?height=400px&classes=shadow)

Click on Edit as Code button, then paste that code:
![as_code](/images/tutorials/npm-audit-parser/as_code.png?height=400px&classes=shadow)

Pipeline is created
![pip_edited](/images/tutorials/npm-audit-parser/pip_edited.png?height=400px&classes=shadow)

{{% /expand%}}


### 5 - Run Workflow

The workflow is now ready to be launched

* Launch the workflow
* Go to pipeline execution
* Click on vulnerability tab

{{%expand "view screenshots..." %}}

Launch the workflow and double click on the pipeline when build finished
![run](/images/tutorials/npm-audit-parser/run_wf.png?height=400px&classes=shadow)

Click on tab 'Vulnerabilities'
![run_vuln](/images/tutorials/npm-audit-parser/run_vulnerability.png?height=400px&classes=shadow)

{{% /expand%}}

### 6 - Application vulnerability

If the workflow has been launch on the default branch of your repository, vulnerabilities are also attached to the CDS application

* Go to Project -> Application tab
* Click on Vulnerabilities tab

{{%expand "view screenshots..." %}}

Go to your project, on application tab
![run](/images/tutorials/npm-audit-parser/project_tab_app.png?height=400px&classes=shadow)

Select your application and go to vulnerabilities tab
![run](/images/tutorials/npm-audit-parser/app_vuln.png?height=400px&classes=shadow)

{{% /expand%}}
