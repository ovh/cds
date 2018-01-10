+++
title = "GitClone"
chapter = true

[menu.main]
parent = "actions-builtin"
identifier = "gitclone"

+++

**GitClone** is a builtin action, you can't modify it.

This action clones a repository into a new directory.

This will retrieve a shallow git clone (`depth==1`).

You may want to use a privateKey to clone from an SSH repository. To do so, you will need to add a project or an application variable of type `key`. `{{.cds.app.a-key.pub}}`

## Parameters

* url - mandatory - the git URL must include information about the transport protocol, the address of the remote server, and the path to the repository.
* privateKey - optional - the private key to be able to git clone from ssh
* user - optional - the user to be able to git clone from https with authentication
* password - optional - the password to be able to git clone from https with authentication
* branch - optional - Instead of pointing the newly created HEAD to the branch pointed to by the cloned repository’s HEAD, point to {{.git.branch}} branch instead.
* commit - optional - the current branch head (HEAD) to the commit
* directory - optional - the name of a directory to clone into.


### Example

* Add repository manager on your application. We can use CDS Variables `{{.git...}}` in Job Configuration

![img](/images/workflows.pipelines.actions.builtin.gitclone-repo-manager.png)

* Job Configuration.

![img](/images/workflows.pipelines.actions.builtin.gitclone-job.png)


* Launch pipeline, check logs

![img](/images/workflows.pipelines.actions.builtin.gitclone-logs.png)

* View artifact

![img](/images/workflows.pipelines.actions.builtin.artifact-upload-view-artifact.png)
