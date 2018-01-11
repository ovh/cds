+++
title = "Sidebar"
weight = 8

+++

The "select box" to select a git branch or a version on UI has been removed. There is only one select box for filter on CDS Tags.

So, what's a tag? A tag is a CDS Variable, exported as a tag. There are default tags as `git.branch`, `git.hash`, `tiggered_by` and environment.

Inside a job, a user can add a Tag with the worker command 

```
worker tag tagKey=tagValue
```

See [worker tab documentation]({{< relref "worker/commands/tag.md" >}})

Tags are useful to add indication on the sidebar about the context of a Run.

You can select the tags displayed on the sidebar Workflow → Advanced → "Tags to display in the sidebar".

![Webhook](/images/workflows.design.sidebar.png)


