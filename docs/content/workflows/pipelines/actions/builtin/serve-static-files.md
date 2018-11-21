+++
title = "Serve Static Files"
chapter = true
+++

**Serve Static Files Action** is a builtin action, you can't modify it.

This action can be used to upload static files and serve them. For example your HTML report about coverage, tests, performances, ...

## Parameters
* name: Name to display in CDS UI and identify your static files
* path: Path where static files will be uploaded (example: mywebsite/*). If it's a file, the entrypoint would be set to this filename by default.
* tag: Filename (and not path) for the entrypoint when serving static files (default: if empty it would be index.html).

### Example

* In this example I created a website with script in bash and use action `Serve Static Files`.

![img](/images/workflows.pipelines.actions.builtin.serve-static-files-job.png)

* Launch pipeline, check logs

![img](/images/workflows.pipelines.actions.builtin.serve-static-files-logs.png)

* View your static files with links in the tab artifact

![img](/images/workflows.pipelines.actions.builtin.serve-static-files-tab.png)
