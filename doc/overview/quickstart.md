## Step 1. Create an account

Go to CDS WebUI and create your account.

In case you are browserphobic you can download the CLI and run
```shell
$ cds user add <username> <email> <fullname>
```
Either way you will receive a confirmation email to activate your account (in case no SMTP is configured, mail is printed in console).

## Step 2. Build and deploy stuff !

Remember: **Every action runs in a different worker**, all build data needed for the next step should be **uploaded as artifact**.

Use CDS templates to bootstrap your applications configurations in 1 minute.

Otherwise, see the ["first pipeline"](/doc/tutorials/first-pipeline.md) tutorial.


## Step 3. Profit

Harness the power of continuous build, test and deployment. Check everything in a single view

![organisation](/doc/img/workflow-view.png)

## Step 4. Go further

Things to try:

 * Setup your own [worker](/doc/overview/worker.md) ([tuto](/doc/tutorials/setup-worker.md))
