## Example

* Add repository manager on your application.

![img](/images/workflows.pipelines.actions.builtin.checkout-application-repo-manager.png)

* Job Configuration.

![img](/images/workflows.pipelines.actions.builtin.checkout-application-edit-job.png)

* Launch workflow, you can select the git branch.

![img](/images/workflows.pipelines.actions.builtin.checkout-application-run-workflow.png)

* View logs on job

![img](/images/workflows.pipelines.actions.builtin.checkout-application-run-job.png)

## Notes

This action clones a repository into a directory. If you want to clone a tag from your repository in this way, in your workflow payload you can add a key in your JSON like `"git.tag": "0.2"`.
