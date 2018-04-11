+++
title = "Build & Deploy a standard application"
weight = 3
+++

{{% notice note %}}

In this tutorial, you will create a CDS Workflow with the Web UI.

* Create a workflow using two pipelines: one for building, another for deploying
* Discover application and environments concepts.
* Discover Run Conditions.

{{% /notice %}}

{{% notice tip %}}
This tutorial seems very long, don't worry, it's very detailed and it will take you about 10-15 minutes to try it.

You will play with many CDS Concepts, discover the reuse of pipelines, how to play with
CDS Variables.
{{% /notice %}}

### 1 - Create your CDS project

Let's create a project. On the top navbar, click on 'Create a project'.

* Enter a Project Name
* The project key will be useful when you want to use [cdsctl]({{< relref "cli/cdsctl/_index.md" >}}).
* Click on 'Create' button.

{{%expand "view screenshots..." %}}

![1_create_prj](/images/getting_started_standard_wf_1_create_prj.png?height=400px)

{{% /expand%}}

### 2 - Add a repository manager on your project

The project is now create, you have to link a repository manager. 
Be sure to have a [Repository manager]({{< relref "/hosting/repositories-manager/_index.md" >}}) setted up on your CDS Instance.

* Select 'Advanced' section
* In the section 'Link to a repository manager', select 'github', then click on 'Connect'
* A popup is displayed, Click on 'Click here' to finalize the link with github. By doing that, you allow CDS to create hook on github.


{{%expand "view screenshots..." %}}

Select 'Advanced' section
![2_admin_prj](/images/getting_started_standard_wf_2_admin_prj.png?height=400px)

In the section 'Link to a repository manager', select 'github', then click on 'Connect'
![3_admin_prj_add_repo](/images/getting_started_standard_wf_3_admin_prj_add_repo.png?height=400px)

A popup is displayed, Click on 'Click here' to finalize the link with github. By doing that, you allow CDS to create hook on github.
![4_admin_prj_add_repo_popup](/images/getting_started_standard_wf_4_admin_prj_add_repo_popup.png?height=400px)

Here, Github is well linked on your CDS project.

![5_admin_repo_linked](/images/getting_started_standard_wf_5_admin_repo_linked.png?height=400px)

{{% /expand%}}

### 3 - Create an application, link it to a Git Repository

You've got a project, linked to Github. Let's create an application.

A CDS Application is useful to have a link to a Git Repository.

* Go on Project -> Applications tab, click on 'Create a new application'
* Enter application name, here 'my-java-app'
* Go on Advanced tab, select a Repository
* Expand configuration, select 'https'. If your repository is public, you can keep empty fields, then click on 'Save'

{{%expand "view screenshots..." %}}

Go on Project -> Applications tab, click on 'Create a new application'
![6_app_tab](/images/getting_started_standard_wf_6_app_tab.png?height=400px)

Enter application name, here 'my-java-app'
![7_create_app](/images/getting_started_standard_wf_7_create_app.png?height=400px)

Go on Advanced tab, select a Repository
![8_admin_app](/images/getting_started_standard_wf_8_admin_app.png?height=400px)

Expand configuration, select 'https'. If your repository is public, you can keep empty fields, then click on 'Save'
![9_app_link_repo](/images/getting_started_standard_wf_9_app_link_repo.png?height=400px)

{{% /expand%}}

### 4 - Add a variable on your application

You've got an application, let's create an application [variable]({{< relref "workflows/pipelines/variables.md" >}}).
We will use it further in a [Job]({{< relref "gettingstarted/concepts/job.md" >}}).

* Select the variables tab, name 'my-variable', type 'string', value 'my-value'
* Then click on 'Save'

{{%expand "view screenshots..." %}}

Select the variables tab, name 'my-variable', type 'string', value 'my-value'
![10_app_create_var](/images/getting_started_standard_wf_10_app_create_var.png?height=400px)

Then click on 'Save'
![11_app_var_created](/images/getting_started_standard_wf_11_app_var_created.png?height=400px)

{{% /expand%}}

### 5 - Create the workflow

Here we go, you will create your first workflow.

* Go to Project -> Worflows tab
* Enter the Workflow name, then click on Next
* You have now to choose the first pipeline. As you don't have a pipeline yet, you will create a new one, named 'build-pipeline'. Click on Next
* Now, you have to select an application. Choose your application 'my-java-app', then click on Next
* We don't need an environment for the build pipeline, Click on 'Finish'

{{%expand "view screenshots..." %}}

Go to Project -> Worflows tab
![12_wf_tab](/images/getting_started_standard_wf_12_wf_tab.png?height=400px)

Enter the Workflow name, then click on Next
![13_create_wf](/images/getting_started_standard_wf_13_create_wf.png?height=400px)

You have now to choose the first pipeline. As you don't have a pipeline yet, you will create a new one, named 'build-pipeline'. Click on Next
![14_create_wf](/images/getting_started_standard_wf_14_create_wf.png?height=400px)

Now, you have to select an application. Choose your application 'my-java-app', then click on Next
![15_create_wf](/images/getting_started_standard_wf_15_create_wf.png?height=400px)

We don't need an environment for the build pipeline, Click on 'Finish'
![16_create_wf](/images/getting_started_standard_wf_16_create_wf.png?height=400px)

The workflow is now created
![17_wf_created](/images/getting_started_standard_wf_17_wf_created.png?height=400px)

{{% /expand%}}

### 6 - Edit your first pipeline for building the application

The workflow is initialized with an empty pipeline named 'build-pipeline'. You have now to create your
first job.

* In your workflow, select the pipeline 'build-pipeline', then click on 'Edit the pipeline'.
* Click on 'Add job'
* Add the first step 'CheckoutApplication'
* Add a second step 'Script'. The content of the script is `mvn package``
* The third step is 'Artifact Upload', to upload your builded binary
* And the last step is 'JUnit'. This step is 'always executed' and let you to see test results on UI.
* Last thing about the 'build-pipeline': as you use mvn, you probably want to add `mvn` and `java`. Click
on Requirements link then add binaries prerequisites.

{{%expand "view screenshots..." %}}

In your workflow, select the pipeline 'build-pipeline', then click on 'Edit the pipeline'. Click on 'Add job'.
![18_edit_build_pipeline](/images/getting_started_standard_wf_18_edit_build_pipeline.png?height=400px)

Add the first step 'CheckoutApplication'
![19_edit_build_pipeline](/images/getting_started_standard_wf_19_edit_build_pipeline.png?height=400px)

Add the first step 'CheckoutApplication'
![20_edit_build_pipeline](/images/getting_started_standard_wf_20_edit_build_pipeline.png?height=400px)

Add a second step 'Script'. The content of the script is `mvn package``
![21_edit_build_pipeline](/images/getting_started_standard_wf_21_edit_build_pipeline.png?height=400px)

The third step is 'Artifact Upload', to upload your builded binary
![22_edit_build_pipeline](/images/getting_started_standard_wf_22_edit_build_pipeline.png?height=400px)

And the last step is 'JUnit'. This step is 'always executed' and let you to see test results on UI.
![23_edit_build_pipeline](/images/getting_started_standard_wf_23_edit_build_pipeline.png?height=400px)

Last thing about the 'build-pipeline': as you use mvn, you probably want to add `mvn` and `java`. Click
on Requirements link then add binaries prerequisites.
![24_edit_build_pipeline_add_requirements](/images/getting_started_standard_wf_24_edit_build_pipeline_add_requirements.png?height=400px)

{{% /expand%}}

### 7 - Add a Hook on your workflow

In this example, we create a Workflow to build & deploy an application. This is a 
standard Continous Integration & Continous Delivery Workflow.

So, we have to trigger this workflow on each commit, on every git branches. This will be 
useful to compile code from all developper and deploy master branch is the buid is Ok.

The application is linked to a Github Git Repository, we have two choice to trigger automatically the workflow:

* add a Git Repository Webhook
* or add a Git Repository Poller.

The difference between both is simple: a Git Repository Webhook does not work if your CDS Instance is not
reacheabled from Github. So, we have to add a Git Repository Poller

* Select the pipeline root, then click on 'Add a hook'
* Choose a Git Repository Poller
* The poller is added and linked to your first pipeline

{{%expand "view screenshots..." %}}

Select the pipeline root, then click on 'Add a hook'
![25_select_root](/images/getting_started_standard_wf_25_select_root.png?height=400px)

Choose a Git Repository Poller
![26_add_repo_poller](/images/getting_started_standard_wf_26_add_repo_poller.png?height=400px)

The poller is added and linked to your first pipeline
![27_hook_poller_added](/images/getting_started_standard_wf_27_hook_poller_added.png?height=400px)

{{% /expand%}}

### 8 - Run your workflow

It's time to launch your Workflow, click on the green button 'Run workflow'.

* On the popup, you can choose the git branch, then click on green button 'OK'
* The first pipeline is building (you can double-click on it, it's a shortcut), you can see logs. 
* The pipeline is done, it's a success.
* Click on 'Test' tab, you can see Unit Tests.
* Click on 'Artifact', you see the builded artifact.

{{%expand "view screenshots..." %}}

On the popup, you can choose the git branch, then click on green button 'OK'
![28_run_wf](/images/getting_started_standard_wf_28_run_wf.png?height=400px)

The first pipeline is building (you can double-click on it, it's a shortcut), you can see logs. 
![29_wf_running](/images/getting_started_standard_wf_29_wf_running.png?height=400px)

The pipeline is done, it's a success.
![30_wf_success](/images/getting_started_standard_wf_30_wf_success.png?height=400px)

Click on 'Test' tab, you can see Unit Tests.
![31_view_tests](/images/getting_started_standard_wf_31_view_tests.png?height=400px)

Click on 'Artifact', you see the builded artifact.
![32_view_artifacts](/images/getting_started_standard_wf_32_view_artifacts.png?height=400px)

{{% /expand%}}

### 9 - Add a pipeline for deploying your application on staging

Ok, we have an artifact to deploy. Let's create a deploy pipeline and trigger it after the build pipeline.

* Select the 'build-pipeline'
* Create a 'deploy-pipeline', then click on 'Next'
* Select the application 'my-java-app', then click on 'Next'
* Create a new environment named 'staging', then click on 'Finish'
* The workflow contains now two pipelines

{{%expand "view screenshots..." %}}

Select the build pipeline
![33_select_root](/images/getting_started_standard_wf_33_select_root.png?height=400px)

Create a 'deploy-pipeline', then click on 'Next'
![34_add_pipeline](/images/getting_started_standard_wf_34_add_pipeline.png?height=400px)

Select the application 'my-java-app', then click on 'Next'
![35_add_pipeline](/images/getting_started_standard_wf_35_add_pipeline.png?height=400px)

Create a new environment named 'staging', then click on 'Finish'
![36_add_pipeline](/images/getting_started_standard_wf_36_add_pipeline.png?height=400px)

The workflow contains now two pipelines
![37_pipeline_added](/images/getting_started_standard_wf_37_pipeline_added.png?height=400px)

{{% /expand%}}

### 10 - Add a pipeline for deploying your application on production

Same as previous, we will add a pipeline to deploy in production.

* Select the 'deploy-pipeline'
* Select the 'deploy-pipeline', then click on 'Next'
* Select the application 'my-java-app', then click on 'Next'
* Create a new environment named 'production', then click on 'Finish'
* The workflow contains now two pipelines

{{%expand "view screenshots..." %}}

Select the 'deploy-pipeline'
![38_pipeline_added](/images/getting_started_standard_wf_38_pipeline_added.png?height=400px)

Select the 'deploy-pipeline', then click on 'Next'
![39_add_pipeline_prod](/images/getting_started_standard_wf_39_add_pipeline_prod.png?height=400px)

Select the application 'my-java-app', then click on 'Next'
![40_add_pipeline_prod](/images/getting_started_standard_wf_40_add_pipeline_prod.png?height=400px)

Create a new environment named 'production', then click on 'Finish'
![41_add_pipeline_prod](/images/getting_started_standard_wf_41_add_pipeline_prod.png?height=400px)

The workflow contains now three pipelines
![42_add_pipeline_prod](/images/getting_started_standard_wf_42_add_pipeline_prod.png?height=400px)

{{% /expand%}}

### 11 - Add run conditions before deploying

So, now, you have a workflow to build your application and deploy it on your staging environment.
But, we don't want to deploy all builds, from all branches, we want to deploy only the 
master branch.
Let's create a Run Condition on `git.branch`, to trigger automatically a deployment on staging if git branch is equals to `master`.

* Select the 'deploy-pipeline', then click on 'Edit run Conditions'
* Add a run condition 'git.branch', with value 'master', click on 'Plus' button
* Click on 'Save'

{{%expand "view screenshots..." %}}

Select the 'deploy-pipeline', then click on 'Edit run Conditions'
![43_select_staging](/images/getting_started_standard_wf_43_select_staging.png?height=400px)

Add a run condition 'git.branch', with value 'master', click on 'Plus' button
![44_edit_run_conditions](/images/getting_started_standard_wf_44_edit_run_conditions.png?height=400px)

Click on 'Save'
![45_edit_run_conditions](/images/getting_started_standard_wf_45_edit_run_conditions.png?height=400px)

{{% /expand%}}

### 12 - Add run conditions before deploying in production

Same as 'deploy-pipeline' on staging, we will add condition on the pipeline which deploy in 'production'.

* Select the 'deploy-pipeline_2', then click on 'Edit run Conditions'
* Add a run condition 'cds.manual', with value 'true', click on 'Plus' button

{{%expand "view screenshots..." %}}


Select the 'deploy-pipeline_2', then click on 'Edit run Conditions'
![52_select_prod](/images/getting_started_standard_wf_52_select_prod.png?height=400px)

Add a run condition 'cds.manual', with value 'true', click on 'Plus' button
![53_run_conditions_prod](/images/getting_started_standard_wf_53_run_conditions_prod.png?height=400px)

{{% /expand%}}


### 13 - Edit the name of the pipelines in your workflow

In your project, you've got two pipelines: 'build-pipeline' and 'deploy-pipeline'.
The 'deploy-pipeline' is used twice: once for 'staging' deploy, another for 'production'.

Let's rename the pipelines on your workflow.

* Select the 'deploy-pipeline', then on the top left, click on on the edit button. Rename to 'auto-deploy-pipeline'
* Do the same for the second 'deploy-pipeline', rename it to 'manual-deploy-pipeline'

{{%expand "view screenshots..." %}}

Select the 'deploy-pipeline', then on the top left, click on on the edit button. Rename to 'auto-deploy-pipeline'
![54_edit_node_name](/images/getting_started_standard_wf_54_edit_node_name.png?height=400px)

Do the same for the second 'deploy-pipeline', rename it to 'manual-deploy-pipeline'
![55_names_edited](/images/getting_started_standard_wf_55_names_edited.png?height=400px)

{{% /expand%}}

### 14 - Edit the 'deploy-pipeline'

The 'deploy-pipeline' is empty for now. Let's add some stuff to simulate a deployment.
We will use [CDS Variable]({{< relref "workflows/pipelines/variables.md" >}}) from application.

* Select the 'auto-deploy-pipeline', then click on 'Edit the pipeline' on the sidebar
* Add a step 'Artifact Download' and a step 'script'.
* The script contains `echo "deploying {{.cds.application}} with variable {{.cds.app.my-variable}} on environment {{.cds.environment}}"`

{{%expand "view screenshots..." %}}

Select the 'auto-deploy-pipeline', then click on 'Edit the pipeline' on the sidebar
![47_edit_pipeline_deploy](/images/getting_started_standard_wf_47_edit_pipeline_deploy.png?height=400px)

Add a step 'Artifact Download' and a step 'script'.
![50_edit_pipeline_deploy](/images/getting_started_standard_wf_50_edit_pipeline_deploy.png?height=400px)

The script contains `echo "deploying {{.cds.application}} with variable {{.cds.app.my-variable}} on environment {{.cds.environment}}"`
![51_edit_pipeline_deploy](/images/getting_started_standard_wf_51_edit_pipeline_deploy.png?height=400px)

{{% /expand%}}


### 15 - Run your workflow

Let's Run the workflow. 

* The pipeline 'auto-deploy-pipeline' is automatically launched.
* The script step on this pipeline contains `deploying my-java-app with variable my-value on environment staging`
* The pipeline deploy in production is not launched, as expected.

{{%expand "view screenshots..." %}}

The pipeline 'auto-deploy-pipeline' is automatically launched.
![57_run](/images/getting_started_standard_wf_57_run.png?height=400px)

The script step on this pipeline contains `deploying my-java-app with variable my-value on environment staging`
![58_view_run_staging](/images/getting_started_standard_wf_58_view_run_staging.png?height=400px)

The pipeline deploy in production is not launched, as expected.
![59_view_run](/images/getting_started_standard_wf_59_view_run.png?height=400px)

{{% /expand%}}

### 16 - Run the deploy in production

As you add a run condition on the 'manual-deploy-pipeline', with `cds.manual = true`, you have to 
click on Run to launch a deployment in production.

* Select the 'manual-deploy-pipeline', then click on the 'Play' button on the top left
* The script step displays `deploying my-java-app with variable my-value on environment production`

{{%expand "view screenshots..." %}}

Select the 'manual-deploy-pipeline', then click on the 'Play' button on the top left
![60_manual_run](/images/getting_started_standard_wf_60_manual_run.png?height=400px)

![62_run_prod](/images/getting_started_standard_wf_62_run_prod.png?height=400px)

The script step displays `deploying my-java-app with variable my-value on environment production`
![63_view_run_prod](/images/getting_started_standard_wf_63_view_run_prod.png?height=400px)

{{% /expand%}}

