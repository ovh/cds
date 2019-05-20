## Example

* In this example I created a website with script in bash and use action `Serve Static Files`. If you want to keep the same URL by .git.branch for example you can indicate in the advanced parameters a `static-key` equals to `{{.git.branch}}`.

![img](/images/workflows.pipelines.actions.builtin.serve-static-files-job.png)

* Launch pipeline, check logs

![img](/images/workflows.pipelines.actions.builtin.serve-static-files-logs.png)

* View your static files with links in the tab artifact

![img](/images/workflows.pipelines.actions.builtin.serve-static-files-tab.png)

## Notes

Pay attention this action is only available if your objectstore is configured to use [Openstack Swift]({{< relref "/docs/integrations/openstack/openstack_swift.md" >}}) or [AWS S3]({{< relref "/docs/integrations/aws/aws_s3.md" >}}). And for now by default your static files will be deleted after 2 months.
