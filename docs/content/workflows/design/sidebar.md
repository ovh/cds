+++
title = "Sidebar"
weight = 8

+++

On the left sidebar, there is only one select box for filter on CDS Tags.

So, what's a tag? A tag is a CDS Variable, exported as a tag. There are default tags as `git.branch`, `git.hash`, `tiggered_by` and environment. For example if you want to know on which branch the build was launched you just have to filter on a specific CDS tag (in this case `git.branch`)

Inside a job, a user can add a Tag with the worker command 

```
worker tag tagKey=tagValue
```

See [worker tab documentation]({{< relref "worker/commands/tag.md" >}})

Tags are useful to add informations and context for a run.

If you want to filter all runs in sidebar, you can select the tags displayed: Go to Workflow → Advanced → "Tags to display in the sidebar".

![Webhook](/images/workflows.design.sidebar.png)


