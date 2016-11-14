## Introduction

CDS can be linked to following repositories manager :

 - **Atlassian Stash**
 - **Github**

It allows you to enable some CDS features such as :

 - Create application in CDS from Stash or Github
 - Attach an application to its Stash or Github repository
 - Fully automatic hook management
 - Branch filtering on application workflows
 - Commit logs on pipeline build details

Go through this tutorial to enable the link between repositories managers and CDS.


You need CDS admin privileges to perform the following steps.
Download and install properly the CDS CLI.

## Authorize CDS on your Stash instance
You need to perform the following steps :

 - Stash admin privileges
 - A RSA Key Pair

### Create a CDS application in Stash
In Stash go to *Administration Settings* / *Application Links*. Create a new Application with :

 - Name : **CDS**
 - Type : **Generic Application**
 - Application URL : *Your CDS URL*
 - Display URL : *Your CDS URL*

On this application, you just have to set up *OAuth Incoming Authentication* :

 - Consumer Key : **CDS**
 - Consumer Name : **CDS**
 - Public Key : *Your CDS RSA public key*
 - Consumer Callback URL : None
 - Allow 2-Legged OAuth : false
 - Execute as : None
 - Allow user impersonation through 2-Legged OAuth : false

### Connect CDS To Stash
With CDS CLI run :

 ```
 $ cds reposmanager add STASH mystash.mynetwork.net http://mystash.mynetwork.net key=privatekey
 ```

And follow instructions.

Set in Vault you CDS **private key** in a secret named : `cds/repositoriesmanager-secrets-mystash.mynetwork.net-privatekey`

Restart CDS.

Now check everything is OK with :
 ```
 $ cds reposmanager list
 ```


## Authorize CDS on Github
### Create a CDS application on Github
Go to `https://github.com/settings/developers` and **Register a new application**: set an application name, the url and a description. Dont set up `Authorization callback URL`.

On the next page Github give you a **Client ID** and a **Client Secret**

### Connect CDS To Github
With CDS CLI run :

 ```
 $ cds reposmanager add GITHUB github.com http://github.com client-id=<your_client_id> client-secret=client-secret
 ```

And follow instructions.

Set in Vault you CDS **Client Secret** in a secret named : `cds/repositoriesmanager-secrets-github.com.net-client-secret`

Restart CDS.

Now check everything is OK with :
 ```
 $ cds reposmanager list
 ```
