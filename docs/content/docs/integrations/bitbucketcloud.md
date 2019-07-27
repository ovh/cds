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

#### VCS µService Configuration

If you don't already have any of vcs integrations on your CDS please follow these steps. The file configuration for the VCS µService can be retreived with:

```bash
$ engine config new vcs > vcs-config.toml

# or with all other configuration parts:
$ engine config new > config.toml
```

Edit the toml file:

- section `[vcs]`
  - the URL will be used by CDS API to reach this µService
  - add a name, as `cds-vcs`. Without a name, the service VCS will not start
  
```toml
[vcs]
  URL = "http://localhost:8084"

  # Name of this CDS VCS Service
  # Enter a name to enable this service
  name = "cds-vcs"
```

- section `[vcs.UI.http]`
  - URL of CDS UI. This URL will be used by Bitbucket Cloud as a callback on Oauth2. This url must be accessible by users' browsers.
  
```toml
    [vcs.UI.http]
      url = "http://localhost:4200"
```

- section `[vcs.api]`
  - this section will be used to communicate with CDS API. Check the url and enter a shared.infra token.
  - Token can be generated with cdsctl: `cdsctl token generate shared.infra persistent`.

```toml
  [vcs.api]
    maxHeartbeatFailures = 10
    requestTimeout = 10
    token = "enter sharedInfraToken from section [api.auth] here"

    [vcs.api.grpc]
      # insecure = false
      url = "http://localhost:8082"

    [vcs.api.http]
      # insecure = false
      url = "http://localhost:8081"
```

Then add this part to specify you want to add bitbucketcloud integration. Set value to `clientId` and `clientSecret`.

```toml
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

#### hooks µService Configuration

If you have not already a hooks µService configured. Then, as the `vcs` µService, you have to configure the `hooks` µService

```bash
$ engine config new hooks > hooks-config.toml
```

In the `[hooks]` section

- check the URL, this will be used by CDS API to call CDS Hooks
- configure `urlPublic` if you want to use [simple Webhook]({{<relref "/docs/concepts/workflow/hooks/webhook.md">}})
- add a name, as `cds-hooks`

In the `[hooks.api]` section

- put the same token as the `[vcs.api]` section


### Start the vcs and hooks µService

*As a CDS Administrator* 

```bash
$ engine start vcs --config vcs-config.toml
$ engine start hooks --config hooks-config.toml

# you can also start CDS api and vcs in the same process:
$ engine start api vcs hooks --config config.toml
```

## Vcs events

For now, CDS supports push events. CDS uses this push event to remove existing runs for deleted branches (24h after branch deletion).
