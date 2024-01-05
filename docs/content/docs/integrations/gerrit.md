---
title: Gerrit Repository Manager
main_menu: true
card: 
  name: repository-manager
---

The Gerrit Repository Manager integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to link a Git Repository hosted by Gerrit
to a CDS application.

This integration enables some features:

 - [Gerrit Hooks]({{<relref "/docs/concepts/workflow/hooks/gerrit.md" >}})
 - Easy to use action [CheckoutApplication]({{<relref "/docs/actions/builtin-checkoutapplication.md" >}}) and [GitClone]({{<relref "/docs/actions/builtin-gitclone.md">}}) for advanced usage
 - Send comments on your Pull-Requests when a workflow is failed
 - Add a vote -1/+1 on a change

## How to configure Gerrit integration

You will have to create 2 users on gerrit: <a href="https://gerrit-review.googlesource.com/Documentation/cmd-create-account.html" target="_blank">[How to]</a>

 - An Administrator User ( with SSH KEY ), to get event from Gerrit Server
 - An User on gerrit ( with httpPassword ), to comment changes with workflow result
 

### Import configuration

Create a yml file:

```yaml
version: v1.0
name: gerrit
type: gerrit
description: "gerrit new dev"
url: https://your-gerrit-instance:9080
auth:
    sshUsername: gerrit-username # # User to access to gerrit event stream
    sshPort: 29418
    sshPrivateKey: -----BEGIN OPENSSH PRIVATE KEY-----\nfoofoofoo\non\none\nline\nhere\n-----END OPENSSH PRIVATE KEY-----" # Private key of the user who access to gerrit event stream
    user: admin # User that review changes
    token: gerrit-generated-password # Http Password of the user that comment changes
```

```sh
cdsctl project vcs import YOUR_CDS_PROJECT_KEY vcs-gerrit.yml
```

See how to generate **[Configuration File]({{<relref "/hosting/configuration.md" >}})**

## Start the vcs µService

```bash
$ engine start vcs

# you can also start CDS api and vcs in the same process:
$ engine start api vcs
```