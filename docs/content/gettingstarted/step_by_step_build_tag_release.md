+++
title = "Build, tag and release an application"
weight = 4
+++

{{% notice note %}}

In this tutorial, you will create a CDS Workflow with the Web UI.

* Create a workflow using two pipelines: one for building, a second for tagging
* You will discover the [GitTag]({{< relref "gittag.md" >}}) action, which create tag compatible which [Semantic Verstionning 2.0.0](https://semver.org/)

{{% /notice %}}

### 1 - Create your CDS project

Let's create a project. On the top navbar, click on 'Create a project'.

* Enter a Project Name
* The project key will be useful when you want to use [cdsctl]({{< relref "cli/cdsctl/_index.md" >}}).
* Click on 'Create' button.

{{%expand "view screenshots..." %}}

![1_create_prj](/images/getting_started_standard_wf_1_create_prj.png?height=400px&classes=shadow)

{{% /expand%}}

### 2 - Add a repository manager on your project

The project is now create, you have to link a repository manager. 
Be sure to have a [Repository manager]({{< relref "/hosting/repositories-manager/_index.md" >}}) setted up on your CDS Instance.

* Select 'Advanced' section
* In the section 'Link to a repository manager', select 'github', then click on 'Connect'
* A popup is displayed, Click on 'Click here' to finalize the link with GitHub. By doing that, you allow CDS to create hook on GitHub.


{{%expand "view screenshots..." %}}

Select 'Advanced' section
![2_admin_prj](/images/getting_started_standard_wf_2_admin_prj.png?height=400px&classes=shadow)

In the section 'Link to a repository manager', select 'github', then click on 'Connect'
![3_admin_prj_add_repo](/images/getting_started_standard_wf_3_admin_prj_add_repo.png?height=400px&classes=shadow)

A popup is displayed, Click on 'Click here' to finalize the link with GitHub. By doing that, you allow CDS to create hook on GitHub.
![4_admin_prj_add_repo_popup](/images/getting_started_standard_wf_4_admin_prj_add_repo_popup.png?height=400px&classes=shadow)

Here, GitHub is well linked on your CDS project.

![5_admin_repo_linked](/images/getting_started_standard_wf_5_admin_repo_linked.png?height=400px&classes=shadow)

{{% /expand%}}

### 3 - Create an application, link it to a Git Repository

You've got a project, linked to GitHub. Let's create an application.

A CDS Application is useful to have a link to a Git Repository.

* Go on Project -> Applications tab, click on 'Create a new application'
* Enter application name, here 'my-java-app'
* Go on Advanced tab, select a Repository
* Expand configuration, select 'https'. If your repository is public, you can keep empty fields, then click on 'Save'

{{%expand "view screenshots..." %}}

Go on Project -> Applications tab, click on 'Create a new application'
![6_app_tab](/images/getting_started_standard_wf_6_app_tab.png?height=400px&classes=shadow)

Enter application name, here 'my-java-app'
![7_create_app](/images/getting_started_standard_wf_7_create_app.png?height=400px&classes=shadow)

Go on Advanced tab, select a Repository
![8_admin_app](/images/getting_started_standard_wf_8_admin_app.png?height=400px&classes=shadow)

Expand configuration, select 'https'. If your repository is public, you can keep empty fields, then click on 'Save'
![9_app_link_repo](/images/getting_started_standard_wf_9_app_link_repo.png?height=400px&classes=shadow)

{{% /expand%}}


### 4 - Generate a GitHub Token

This token will be used to let CDS create a tag.

* Go on https://github.com/settings/tokens/new, enter a description. Example 'cds-demo', click on Generate Token.
* Go on CDS, select your application and put the token in field password

{{%expand "view screenshots..." %}}

Create a token on GitHub.
![10_github](/images/getting_started_build_tag_wf_10_github.png?height=400px&classes=shadow)

GitHub give you a token, put it in password field.
![11_set_token](/images/getting_started_build_tag_wf_11_set_token.png?height=400px&classes=shadow)

{{% /expand%}}

### 5 - Create the workflow

* Go to Project -> Workflows tab
* Enter the Workflow name, then click on Next
* You have now to choose the first pipeline. As you don't have a pipeline yet, you will create a new one, named 'build-pipeline'. Click on Next
* Now, you have to select an application. Choose your application 'my-java-app', then click on Next
* We don't need an environment, neither platform for the build pipeline, Click on 'Finish'

Notice: the build pipeline does nothing here. You can add some job inside it, please read [this tutorial]({{< relref "step_by_step_build_deploy.md" >}})
to create a 'build' pipeline.

{{%expand "view screenshots..." %}}

Create the pipeline
![12_create_pipeline](/images/getting_started_build_tag_wf_12_create_pipeline.png?height=400px&classes=shadow)

Then select your application.
![13_select_app](/images/getting_started_build_tag_wf_13_select_app.png?height=400px&classes=shadow)

Click on create to create the workflow.
![14_create](/images/getting_started_build_tag_wf_14_create.png?height=400px&classes=shadow)


{{% /expand%}}

### 6 - Add a Hook on your workflow

In this example, we create a Workflow to build, tag and release an application. 

So, we have to trigger this workflow on each commit, on every git branches - and on each tag created. This will be 
useful to compile code from all developer and sometimes create a tag from master branch if the build is Success.

The application is linked to a GitHub Git Repository, we have two choices to trigger automatically the workflow:

* add a Git Repository Webhook
* or add a Git Repository Poller.

The difference between both is simple: a Git Repository Webhook does not work if your CDS Instance is not
reachable from GitHub. So, we have to add a Git Repository Poller

* Select the pipeline root, then click on 'Add a hook'
* Choose a Git Repository Poller
* The poller is added and linked to your first pipeline

{{%expand "view screenshots..." %}}

![15_add_hook](/images/getting_started_build_tag_wf_15_add_hook.png?height=400px&classes=shadow)

{{% /expand%}}

### 7 - Create a tag

* Select the pipeline 'build-pipeline', then click on the sidebar 'Add a pipeline'
* Create a new pipeline named 'create-tag', then select the application 'my-java-app'
* We don't need an environment, neither platform for create a tag, Click on 'Finish'
* Edit the pipeline 'create-tag'
* Click on Edit as Code button, then paste that code:

```yml
version: v1.0
name: create-tag
parameters:
  tagLevel:
    type: list
    default: major;minor;patch
    description: major, minor or patch
jobs:
- job: CreateTag
  steps:
  - checkout: '{{.cds.workspace}}'
  - gitTag:
      path: '{{.cds.workspace}}'
      tagLevel: '{{.cds.pip.tagLevel}}'
      tagMessage: Release from CDS run {{.cds.version}}
```

{{%expand "view screenshots..." %}}

Select the pipeline 'build-pipeline', then click on the sidebar 'Add a pipeline'
![16_add_pipeline](/images/getting_started_build_tag_wf_16_add_pipeline.png?height=400px&classes=shadow)

Edit the pipeline 'create-tag'
![17_view](/images/getting_started_build_tag_wf_17_view.png?height=400px&classes=shadow)

Click on Edit as Code button, then paste that code:
![18_create_pipeline_as_code](/images/getting_started_build_tag_wf_18_create_pipeline_as_code.png?height=400px&classes=shadow)

Pipeline is created
![19_view_pipeline](/images/getting_started_build_tag_wf_19_view_pipeline.png?height=400px&classes=shadow)

{{% /expand%}}

### 8 - Run Workflow

The workflow is now ready to be launched, but before launch it, let's configure 
some Run Condition on the pipeline 'create-tag'. We don't want to launch it on 
each commit - we want to decide when to launch it.

* Click on the pipeline 'create-tag'
* Add two Run Conditions:
    * cds.manual = true
    * git.branch = master
* Launch the workflow, select the tag level, then click on Run

{{%expand "view screenshots..." %}}

Edit Run Conditions.
![20_edit_run_conditions](/images/getting_started_build_tag_wf_20_edit_run_conditions.png?height=400px&classes=shadow)

Launch the workflow.
![21_launch](/images/getting_started_build_tag_wf_21_launch.png?height=400px&classes=shadow)

The workflow is stopped, because you set cds.manual to true in your run conditions.
![22_launch_view](/images/getting_started_build_tag_wf_22_launch_view.png?height=400px&classes=shadow)

Select the pipeline, then click on the 'play' button.
![23_launch_view](/images/getting_started_build_tag_wf_23_launch_view.png?height=400px&classes=shadow)

You can choose the tag level.
![24_launch_create_tag](/images/getting_started_build_tag_wf_24_launch_create_tag.png?height=400px&classes=shadow)

Tag is created, cf. step logs.
![25_launch_tag_created](/images/getting_started_build_tag_wf_25_launch_tag_created.png?height=400px&classes=shadow)

Tag is created on GitHub.
![26_tag_created_github](/images/getting_started_build_tag_wf_26_tag_created_github.png?height=400px&classes=shadow)

{{% /expand%}}

### 9 - Release Action

[Release action]({{< relref "release.md" >}}) action is implemented for GitHub only. 
You can use it to create a release from a tag and push some artifacts on it.

{{%expand "view screenshots..." %}}

![27_release_action](/images/getting_started_build_tag_wf_27_release_action.png?height=400px&classes=shadow)

{{% /expand%}}
