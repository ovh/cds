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
 

### Complete CDS Configuration File

```yaml
 [vcs.servers]
     [vcs.servers.gerrit]
       # URL of Gerrit
       url = "http://localhost:9080"
 
       [vcs.servers.gerrit.gerrit]
         # Disable event listener
         disableGerritEvent = false
 
         # SSH port of gerrit
         sshport = 29418
 
         [vcs.servers.gerrit.gerrit.EventStream]
           # Private key of the user who access to gerrit event stream
           privateKey = "XXXXXXX"
 
           # User to access to gerrit event stream
           user = "admin"
 
         [vcs.servers.gerrit.gerrit.Reviewer]
           # Http Password of the user that comment changes
           token = "XXXXXXXXXX"
 
           # User that review changes
           user = "Verifier"

```

See how to generate **[Configuration File]({{<relref "/hosting/configuration.md" >}})**

## Start the vcs µService

```bash
$ engine start vcs

# you can also start CDS api and vcs in the same process:
$ engine start api vcs
```