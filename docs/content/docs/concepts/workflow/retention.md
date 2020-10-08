---
title: "Retention"
weight: 10
---

You can configure the workflow run retention in the workflow advanced section on the CDS UI.

![retention.png](../images/workflow_retention.png)


* The first line defines the number maximum of runs that CDS can keep for this workflow. Only a CDS administrator can update this value.

* On the second line, you will be able to define your retention policy through a lua condition.
You will be able to use these variables:
  * run_days_before: to identify runs older than x days
  * git_branch_exist: to identify if the git branch used for this run still exists on the git repository
  * run_status: to identidy run status
  * and all variables defined in your workflow payload

For example, the rule defined above means:

Keep workflow run for 365 days, but if branch does not exist on repository, only keep the run for 2 days.
 

* The dry run button allows you to test your lua expression. The result is a table filled with all runs that would be kept
