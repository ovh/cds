+++
title = "Gitlab"
weight = 2

+++

## Authorize CDS on your Gitlab instance
What you need to perform the following steps :

 - Gitlab admin privileges

### Create a CDS application on Gitlab
In Gitlab go to *Settings* / *Application* section. Create a new application with :

 - Name : **CDS**
 - Redirect URI : **http(s)://<your-cds-api>/repositories_manager/oauth2/callback**

Scopes :

 - API
 - read_user
 - read_registry

### Connect CDS to Gitlab
Using CDS CLI, run :

 ```
 $ cds admin reposmanager add GITLAB mygitlab.mynetwork.net http://mygitlab.mynetwork.net app-id=gitlabappid
 ```

And follow instructions.

### Update config.toml and restart

Update the secret value in `api.vcs.gitlab` section then restart CDS.


You can check operation has succeeded with :

 ```
 $ cds admin reposmanager list
 ```
