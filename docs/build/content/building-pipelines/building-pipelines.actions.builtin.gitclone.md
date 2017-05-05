+++
title = "GitClone"
chapter = true

[menu.main]
parent = "actions-builtin"
identifier = "gitclone"

+++

**GitClone** is a builtin action, you can't modify it.

This action clone a repository into a new directory.

Git Clone will be done with a `depth` of 1.

You can use a privateKey, this is usually a project or application variable of type `key`. `{{.cds.app.a-key.pub}}`

## Parameters

* url - mandatory - URL must contain information about the transport protocol, the address of the remote server, and the path to the repository.
* privateKey - optional - the private key to be able to git clone from ssh
* user - optional - the user to be able to git clone from https with authentication
* password - optional - the password to be able to git clone from https with authentication
* branch - optional - Instead of pointing the newly created HEAD to the branch pointed to by the cloned repository’s HEAD, point to {{.git.branch}} branch instead.
* commit - optional - the current branch head (HEAD) to the commit
* directory - optional - the name of a directory to clone into.


### Example

* Add repository manager on your application. We can use CDS Variables `{{.git...}}` in Job Configuration

![img](/images/building-pipelines.actions.builtin.gitclone-repo-manager.png)

* Job Configuration.

![img](/images/building-pipelines.actions.builtin.gitclone-job.png)


* Launch pipeline, check logs

![img](/images/building-pipelines.actions.builtin.gitclone-logs.png)

* View artifact

![img](/images/building-pipelines.actions.builtin.artifact-upload-view-artifact.png)
