## Example

* Add repository manager on your application.

![img](/images/workflows.pipelines.actions.builtin.gitclone-repo-manager.png)

* Job Configuration.

![img](/images/workflows.pipelines.actions.builtin.gitclone-edit-job.png)

* Launch workflow, you can select the git branch.

![img](/images/workflows.pipelines.actions.builtin.gitclone-run-workflow.png)

* View logs on job

![img](/images/workflows.pipelines.actions.builtin.gitclone-run-job.png)

## Notes

By default, depth is 50 and git clone with `--single-branch` automatically.
So, if you want to do in a step script `git diff anotherBranch`, you have to set depth to 'false'.

If there is no user && password && sshkey set in action GitClone, CDS checks on Application VCS Strategy if some auth parameters can be used.
