+++
title = "Error management"
weight = 1

+++

This page explains how to deal with errors in CDS code. Error returned from CDS contains a message, an HTTP status code, a stack trace and a unique id.

Errors can be forwarded to a Graylog instance then retrieved directly from the ctl (see api.graylog and log.graylog sections in cds configuration file to setup).
```bash
cdsctl admin errors get <error_uuid>
```

## Usage in code

All errors from lib should be wrapped like **sdk.WithStack(err)** or **sdk.WrapError(err, format, values...)** directly when created. 
```go
if err := json.Unmarshal(...); err != nil {
    return sdk.WithStack(err) // or return sdk.WrapError(err, "Cannot unmarshal given data")
}
```

**WrapError** can be used to add more details about an error when returned.
```go
func one() error { return sdk.WithStack(json.Unmarshal(...)) }

func two() error { return sdk.WrapError(one(), "Error calling one") }

func three() error { return sdk.WrapError(two(), "Error calling two") }
```

If the error was already wrapped an not more info is needed you should run it directly.
```go
func four() error {
    if err := three(); err != nil {
        return err
    }
    ...
}
```

To create an error that will generate a specific HTTP status code you should use the **sdk.NewError** func or returned an existing sdk.Error.
```go
if err := json.Unmarshal(...); err != nil {
    return sdk.NewError(sdk.ErrWrongRequest, err) // returns a 400 http code with default translated message and from value that contains err cause. 
}

if err := json.Unmarshal(...); err != nil {
    return sdk.NewErrorFrom(sdk.ErrWrongRequest, "A text that will be in from message") // returns a 400 http code with default translated message and test as from. 
}

if err := json.Unmarshal(...); err != nil {
    return sdk.WrapError(sdk.ErrWrongRequest, "Cannot unmarshal given data") // or return sdk.WithStack(sdk.ErrWrongRequest) returns a 400 http code with default translated message.
}
```

To compare if an error match a existing sdk.Err use the **sdk.ErrorIs** func, using equality operator will not work if the error was wrapped.
A not wrapped lib error will match sdk.ErrUnknownError (to check if error is unknown you can use sdk.ErrorIsunknown).
```go
if err := one(); err != nil {
    if sdk.ErrorIs(err, sdk.ErrNotFound) {
        // do something specific for not found error
    }
}

err := json.Unmarshal(...)
sdk.ErrorIs(err, sdk.ErrUnknownError) => true
sdk.ErrorIsUnknown(err) => true
```

To check if an error root cause is equal to a known library error you could use the **sdk.Cause** func.
```go
err := sdk.WrapError(sdk.WrapError(sql.ErrNoRows, "The error is now wrapped"), "Add more info on the error") 
err == sql.ErrNoRows => false
sdk.Cause(err) == sql.ErrNoRows => true
```
