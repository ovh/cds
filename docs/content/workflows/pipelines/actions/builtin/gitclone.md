+++
title = "GitClone"
chapter = true

+++

**GitClone** is a builtin action, you can't modify it.

This action clones a repository into a new directory.

## Parameters

* url - mandatory - the git URL must include information about the transport protocol, the address of the remote server, and the path to the repository.
* privateKey - optional - the private key to be able to git clone from ssh
* user - optional - the user to be able to git clone from https with authentication
* password - optional - the password to be able to git clone from https with authentication
* branch - optional - Instead of pointing the newly created HEAD to the branch pointed to by the cloned repository’s HEAD, point to {{.git.branch}} branch instead.
* commit - optional - the current branch head (HEAD) to the commit
* directory - optional - the name of a directory to clone into.

Advanced parameters:

* depth - optional - 50 by default. You can remove --depth with the value 'false'
* submodules - true by default, you can set false to avoid this.
* tag - optional - empty by default, you can set to `{{.git.tag}}` to clone a tag from your repository. In this way, in your workflow payload you can add a key in your JSON like `"git.tag": "0.2"`.

Notes:

By default, depth is 50 and git clone with `--single-branch` automatically.
So, if you want to do in a step script `git diff anotherBranch`, you have to set depth to 'false'.

If there is no user && password && sshkey setted in action GitClone, CDS checks on Application VCS Strategy if some auth parameters can be used.


### Example

* Add repository manager on your application.

![img](/images/workflows.pipelines.actions.builtin.gitclone-repo-manager.png)

* Job Configuration.

![img](/images/workflows.pipelines.actions.builtin.gitclone-edit-job.png)

* Launch workflow, you can select the git branch.

![img](/images/workflows.pipelines.actions.builtin.gitclone-run-workflow.png)

* View logs on job

![img](/images/workflows.pipelines.actions.builtin.gitclone-run-job.png)
