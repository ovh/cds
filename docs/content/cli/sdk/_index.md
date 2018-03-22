+++
title = "SDK Golang"
+++

## Generate a persitent token

There is three ways to generate a persistent token:

- [cdsctl login]({{< relref "/cli/cdsctl/login.md" >}})
- [call CDS API]({{< relref "/cli/api/_index.md" >}})
- Code it:

```go
conf := cdsclient.Config{
    Host:    "http://your-cds-instance",
}

client = cdsclient.New(conf)
// replace username and password
ok, token, err := client.UserLogin(username, password)
if err != nil {
    return err
}
if !ok {
    return fmt.Errorf("login failed")
}

fmt.Printf("export CDS_API_URL=%s\n", url)
fmt.Printf("export CDS_USER=%s\n", username)
fmt.Printf("export CDS_TOKEN=%s\n", token)

```

## How to use it?

You have to initialize a cdsclient:

```go
cfg := cdsclient.Config{
    Host:  host,
    Token: token,
    User:  username,
}
client := cdsclient.New(cfg)
```

and then, you can use it:

```go

// list workers
workers, err := client.WorkerList()

// list users
users, err := client.UserList()

// list workflow runs
runs, err := client.client.WorkflowRunList(...)

```

Go on https://godoc.org/github.com/ovh/cds/sdk/cdsclient to see all available funcs.
	

## See also

{{%children style="ul"%}}