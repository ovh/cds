+++
title = "Building CDS Workflows"
weight = 3

+++

A CDS Workflow is composed of pipelines and can use some features as join, hooks, mutex, payload... 
You can create a CDS Workflow with the web UI, you can also create a workflow with the command line [cdsctl]({{< relref "cli/cdsctl/_index.md" >}}).

A pipeline is composed of stages and jobs. You can create a pipeline with the web UI or you can import it with [cdsctl]({{< relref "cli/cdsctl/_index.md" >}}).

### Use cdsctl

* [cdsctl workflow import]({{< relref "cli/cdsctl/workflow/import.md" >}})
* [cdsctl workflow export]({{< relref "cli/cdsctl/workflow/export.md" >}})
* [cdsctl workflow pull]({{< relref "cli/cdsctl/workflow/pull.md" >}})
* [cdsctl workflow push]({{< relref "cli/cdsctl/workflow/push.md" >}})
* [cdsctl pipeline import]({{< relref "cli/cdsctl/pipeline/import.md" >}})
* [cdsctl pipeline export]({{< relref "cli/cdsctl/pipeline/export.md" >}})

### Use CDS WEB UI
{{%children style=""%}}
