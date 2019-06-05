---
title: Bitbucket Cloud
main_menu: true
card: 
  name: repository-manager
---

The Bitbucket Cloud Integration have to be configured on your CDS by a CDS Administrator.

This integration allows you to link a Git Repository hosted by your Bitbucket Cloud
to a CDS Application.

This integration enables some features:

 - [Git Repository Webhook]({{<relref "/docs/concepts/workflow/hooks/git-repo-webhook.md" >}})
 - Easy to use action [CheckoutApplication]({{<relref "/docs/actions/builtin-checkoutapplication.md" >}}) and [GitClone]({{<relref "/docs/actions/builtin-gitclone.md">}}) for advanced usage
 - Send [build notifications](https://confluence.atlassian.com/bitbucket/check-build-status-in-a-pull-request-945541505.html) on your Pull-Requests and Commits on Bitbucket Cloud

## How to configure Bitbucket Cloud integration

+ Go on [Bitbucket.org](https://bitbucket.org/dashboard/overview) and log in.
+ From your avatar in the bottom left, click ***Bitbucket settings***.
+ Click OAuth from the left navigation.
+ Click the Add consumer button.
+ Bitbucket requests some informations: for the `name` you can simply write CDS, `description` is optional, `callback url` must be the URL of your CDS -> {CDS_UI_URL}/cdsapi/repositories_manager/oauth2/callback (if you are in development mode you have to omit /cdsapi and replace {CDS_UI_URL} with your API URL), `URL` is optional.
+ Click on Save and toggle the consumer name to see the generated `Key` and `Secret`. It correspond to `clientId` and `clientSecret` in the CDS config.toml file.

### Complete CDS Configuration File

Set value to `privateKey`. You can modify `consumerKey` if you want.

```yaml
 [vcs.servers]
    [vcs.servers.bitbucketcloud]

      [vcs.servers.bitbucketcloud.bitbucketcloud]

        # Bitbucket Cloud OAuth Key
        clientId = "XXXX"

        # Bitbucket Cloud OAuth Secret
        clientSecret = "XXXX"

        # Does webhooks are supported by VCS Server
        disableWebHooks = false

        # Does webhooks creation are supported by VCS Server
        disableWebHooksCreation = false

        #proxyWebhook = "https://myproxy.com/"

        [vcs.servers.bitbucketcloud.bitbucketcloud.Status]

          # Set to true if you don't want CDS to push statuses on the VCS server
          disable = false

          # Set to true if you don't want CDS to push CDS URL in statuses on the VCS server
          showDetail = false
```

See how to generate **[Configuration File]({{<relref "/hosting/configuration.md" >}})**

## Start the vcs µService

```bash
$ engine start vcs

# you can also start CDS api and vcs in the same process:
$ engine start api vcs
```

## Vcs events

For now, CDS supports push events. CDS uses this push event to remove existing runs for deleted branches (24h after branch deletion).
