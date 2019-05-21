## Example

* Workflow Configuration: a pipeline doing an `upload artifact` and another doing a `download artifact`.

![img](../images/artifact-download-workflow.png)

* Run pipeline, check logs

![img](../images/artifact-download-logs.png)

## Worker Download Command

You can download an artifact with the built-in action - or use the worker command.

Example of a step script using [worker download command]({{< relref "/docs/components/worker/download.md" >}})

![img](../images/artifact-worker-download.png)
