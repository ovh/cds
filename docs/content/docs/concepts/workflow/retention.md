---
title: "Retention"
weight: 10
---

Information about workflow execution retention can be found in the workflow advanced section in the CDS UI

![retention.png](../images/workflow_retention.png)


* The first line, define the number maximum of execution that CDS can keep for this workflow. Only a CDS administrator can update this value

* On the second line, you will be able to define your retention policy through a lua condition.
You will be able to use these variables:
  * run_date_before: To identify execution older than x days
  * git_branch_exist: To identify if the execution git branch still exist on repository side
  * run_status: To identidy execution status
  * and all variables defined in your workflow payload

For example, the rule defined above means:

Keep workflow execution for 365 days, but if branch does not exist on repository, only keep the execution 2 days.
 

* The dry run button allow you to test your lua expression. The result is a table filled with all executions that would be kept
