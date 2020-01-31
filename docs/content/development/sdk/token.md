---
title: "Token creation"
weight: 8
card: 
  name: rest-sdk
---

There are two types of token in CDS:

 - **signin token**: can also be named "token" when using CDS command line. This token is like a scoped "password" and can be used to sign-in to CDS. You can generate a sign-in token with the web ui or with CDS command line.

 - **session token**: you should not manipulate it directly as its life is limited. This token is used to authenticate an API call and will be created when you successfully sign-in to CDS.


## Generate a sign-in token

You will be able to generate a new sign-in token for a builtin consumer with the web UI or the command line.

### With the WEB UI
+ Go in Settings > Profile > Authentication

![cds ui profile page](/images/ui_profile_page.png)

+ Then click on `+` at the right of `My consumers` to open modal, then click create to obtain a sign-in token.

![cds ui consumer creation](/images/ui_create_consumer.png)

### With CDS command line

{{< note >}}
To create a builtin consumer you should first be signed into CDS using local authentication for example.
{{< /note >}}

```txt
$ cdsctl consumer new
? Name my-bot
? Description A bot consumer to import my templates
? Select groups availables for the new consumer my-group
? Select scopes availables for the new consumer Template
Builtin consumer successfully created, use the following token to sign in:
<signin-token-value>
```

## Generate a session token

Sometimes if you want to call CDS through its APIs you will have to sign-in to obtain a session token like the following:
```sh
curl -X POST -d '{"token":"<signin-token-value>"}' http://my-cds/auth/consumer/builtin/signin
``` 
You will get a response that contains a session token, the session token is also set as a cookie in the response.
```json
{
  "api_url":"http://my-cds",
  "token":"<session-token-value>",
  "user": {
    "id":"my-user-uuid",
    "created":"...",
    "username":"my-username",
    "fullname":"My Fullname",
    "ring":"USER"
  }
}
```
