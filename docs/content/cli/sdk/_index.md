+++
title = "SDK Golang"
+++

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