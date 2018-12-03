+++
title = "Serve Static Files"
chapter = true
+++

**Serve Static Files Action** is a builtin action, you can't modify it.

This action can be used to upload static files and serve them. For example your HTML report about coverage, tests, performances, ...

Pay attention this action is only available if your objectstore is configured to use Openstack Swift. And fow now by default your static files will be deleted after 2 months.

## Parameters
* name: Name to display in CDS UI and identify your static files
* path: Path where static files will be uploaded (example: mywebsite/*). If it's a file, the entrypoint would be set to this filename by default.
* entrypoint: Filename (and not path) for the entrypoint when serving static files (default: if empty it would be index.html).
* static-key: Indicate a static-key which will be a reference to keep the same generated URL. Example: {{.git.branch}}.

### Example

* In this example I created a website with script in bash and use action `Serve Static Files`. If you want to keep the same URL by .git.branch for example you can indicate in the advanced parameters a `static-key` equals to `{{.git.branch}}`.

![img](/images/workflows.pipelines.actions.builtin.serve-static-files-job.png)

* Launch pipeline, check logs

![img](/images/workflows.pipelines.actions.builtin.serve-static-files-logs.png)

* View your static files with links in the tab artifact

![img](/images/workflows.pipelines.actions.builtin.serve-static-files-tab.png)
