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
This tutorial seems very long, don't worry, it will take you about 10-15 minutes to try it.

You will play with many CDS Concepts, discover the reuse of pipelines, how to play with
CDS Variables.
{{% /notice %}}

### 1 - Create your project

{{%expand "view..." %}}

![Welcome Page](/images/getting_started_standard_wf_0_welcome.png)
![1_create_prj](/images/getting_started_standard_wf_1_create_prj.png)

{{% /expand%}}

### 2 - Add a repository manager on your project

{{%expand "view..." %}}
![2_admin_prj](/images/getting_started_standard_wf_2_admin_prj.png)
![3_admin_prj_add_repo](/images/getting_started_standard_wf_3_admin_prj_add_repo.png)
![4_admin_prj_add_repo_popup](/images/getting_started_standard_wf_4_admin_prj_add_repo_popup.png)
![5_admin_repo_linked](/images/getting_started_standard_wf_5_admin_repo_linked.png)

{{% /expand%}}

### 3 - Create an application, link it to a Git Repository

{{%expand "view..." %}}

![6_app_tab](/images/getting_started_standard_wf_6_app_tab.png)
![7_create_app](/images/getting_started_standard_wf_7_create_app.png)
![8_admin_app](/images/getting_started_standard_wf_8_admin_app.png)
![9_app_link_repo](/images/getting_started_standard_wf_9_app_link_repo.png)
{{% /expand%}}

### 4 - Add a variable on your application

{{%expand "view..." %}}

![10_app_create_var](/images/getting_started_standard_wf_10_app_create_var.png)
![11_app_var_created](/images/getting_started_standard_wf_11_app_var_created.png)

{{% /expand%}}

### 5 - Create the workflow

{{%expand "view..." %}}

![12_wf_tab](/images/getting_started_standard_wf_12_wf_tab.png)
![13_create_wf](/images/getting_started_standard_wf_13_create_wf.png)
![14_create_wf](/images/getting_started_standard_wf_14_create_wf.png)
![15_create_wf](/images/getting_started_standard_wf_15_create_wf.png)
![16_create_wf](/images/getting_started_standard_wf_16_create_wf.png)
![17_wf_created](/images/getting_started_standard_wf_17_wf_created.png)

{{% /expand%}}

### 6 - Edit your first pipeline for building the application

{{%expand "view..." %}}

![18_edit_build_pipeline](/images/getting_started_standard_wf_18_edit_build_pipeline.png)
![19_edit_build_pipeline](/images/getting_started_standard_wf_19_edit_build_pipeline.png)
![20_edit_build_pipeline](/images/getting_started_standard_wf_20_edit_build_pipeline.png)
![21_edit_build_pipeline](/images/getting_started_standard_wf_21_edit_build_pipeline.png)
![22_edit_build_pipeline](/images/getting_started_standard_wf_22_edit_build_pipeline.png)
![23_edit_build_pipeline](/images/getting_started_standard_wf_23_edit_build_pipeline.png)
![24_edit_build_pipeline_add_requirements](/images/getting_started_standard_wf_24_edit_build_pipeline_add_requirements.png)

{{% /expand%}}

### 7 - Add a Hook on your workflow

{{%expand "view..." %}}

![25_select_root](/images/getting_started_standard_wf_25_select_root.png)
![26_add_repo_poller](/images/getting_started_standard_wf_26_add_repo_poller.png)
![27_hook_poller_added](/images/getting_started_standard_wf_27_hook_poller_added.png)

{{% /expand%}}

### 8 - Run your workflow

{{%expand "view..." %}}

![28_run_wf](/images/getting_started_standard_wf_28_run_wf.png)
![29_wf_running](/images/getting_started_standard_wf_29_wf_running.png)
![30_wf_success](/images/getting_started_standard_wf_30_wf_success.png)
![31_view_tests](/images/getting_started_standard_wf_31_view_tests.png)
![32_view_artifacts](/images/getting_started_standard_wf_32_view_artifacts.png)

{{% /expand%}}

### 9 - Add a pipeline for deploying your application on staging

{{%expand "view..." %}}

![33_select_root](/images/getting_started_standard_wf_33_select_root.png)
![34_add_pipeline](/images/getting_started_standard_wf_34_add_pipeline.png)
![35_add_pipeline](/images/getting_started_standard_wf_35_add_pipeline.png)
![36_add_pipeline](/images/getting_started_standard_wf_36_add_pipeline.png)
![37_pipeline_added](/images/getting_started_standard_wf_37_pipeline_added.png)
![38_pipeline_added](/images/getting_started_standard_wf_38_pipeline_added.png)

{{% /expand%}}

### 10 - Add a pipeline for deploying your application on production

{{%expand "view..." %}}

![39_add_pipeline_prod](/images/getting_started_standard_wf_39_add_pipeline_prod.png)
![40_add_pipeline_prod](/images/getting_started_standard_wf_40_add_pipeline_prod.png)
![41_add_pipeline_prod](/images/getting_started_standard_wf_41_add_pipeline_prod.png)
![42_add_pipeline_prod](/images/getting_started_standard_wf_42_add_pipeline_prod.png)

{{% /expand%}}

### 11 - Add run condition before deploying

{{%expand "view..." %}}

![43_select_staging](/images/getting_started_standard_wf_43_select_staging.png)
![44_edit_run_conditions](/images/getting_started_standard_wf_44_edit_run_conditions.png)
![45_edit_run_conditions](/images/getting_started_standard_wf_45_edit_run_conditions.png)
![46_edit_run_conditions](/images/getting_started_standard_wf_46_edit_run_conditions.png)

{{% /expand%}}

### 12 - Edit the 'deploy' pipeline

{{%expand "view..." %}}

![47_edit_pipeline_deploy](/images/getting_started_standard_wf_47_edit_pipeline_deploy.png)
![48_edit_pipeline_deploy](/images/getting_started_standard_wf_48_edit_pipeline_deploy.png)
![49_edit_pipeline_deploy](/images/getting_started_standard_wf_49_edit_pipeline_deploy.png)
![50_edit_pipeline_deploy](/images/getting_started_standard_wf_50_edit_pipeline_deploy.png)
![51_edit_pipeline_deploy](/images/getting_started_standard_wf_51_edit_pipeline_deploy.png)

{{% /expand%}}

### 13 - Add run condition before deploying in production

{{%expand "view..." %}}

![52_select_prod](/images/getting_started_standard_wf_52_select_prod.png)
![53_run_conditions_prod](/images/getting_started_standard_wf_53_run_conditions_prod.png)

{{% /expand%}}

### 14 - Edit the name of the pipelines in your workflow

{{%expand "view..." %}}

![54_edit_node_name](/images/getting_started_standard_wf_54_edit_node_name.png)
![55_names_edited](/images/getting_started_standard_wf_55_names_edited.png)

{{% /expand%}}

### 15 - Run your workflow

{{%expand "view..." %}}

![56_run](/images/getting_started_standard_wf_56_run.png)
![57_run](/images/getting_started_standard_wf_57_run.png)
![58_view_run_staging](/images/getting_started_standard_wf_58_view_run_staging.png)
![59_view_run](/images/getting_started_standard_wf_59_view_run.png)

{{% /expand%}}

### 16 - Run the deploy in production

{{%expand "view..." %}}

![60_manual_run](/images/getting_started_standard_wf_60_manual_run.png)
![61_manual_run](/images/getting_started_standard_wf_61_manual_run.png)
![62_run_prod](/images/getting_started_standard_wf_62_run_prod.png)
![63_view_run_prod](/images/getting_started_standard_wf_63_view_run_prod.png)

{{% /expand%}}

### 17 - Advanced section - edit tags on the workflow sidebar

{{%expand "view..." %}}

![65_edit_tags](/images/getting_started_standard_wf_65_edit_tags.png)
![66_edit_tags](/images/getting_started_standard_wf_66_edit_tags.png)

{{% /expand%}}

### 18 - Manage workflow notifications

{{%expand "view..." %}}

![67_add_notifs](/images/getting_started_standard_wf_67_add_notifs.png)
![67_view_tags](/images/getting_started_standard_wf_67_view_tags.png)
![68_notif_added](/images/getting_started_standard_wf_68_notif_added.png)

{{% /expand%}}
