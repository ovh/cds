+++
title = "Error management"
weight = 1

+++

This page explains how to deal with errors in CDS code. Error returned from CDS contains a message, an HTTP status code and an error unique id that can be used to retrieve the error stack trace with the ctl.
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

func three() error { return sdk.WrapError(one(), "Error calling two") }
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
    return sdk.NewError(sdk.ErrWrongRequest, err) // returns a 400 http code with translated message and cause. 
}

if err := json.Unmarshal(...); err != nil {
    return sdk.WrapError(sdk.ErrWrongRequest, "Cannot unmarshal given data") // returns a 400 http code with translated message and cause. 
}

if err := json.Unmarshal(...); err != nil {
    return sdk.WithStack(sdk.ErrWrongRequest) // returns a 400 http code with translated message but no info about the source error. 
}
```

To compare if an error match a existing sdk.Err use the **sdk.ErrorIs** func, using equality operator will not work if the error was wrapped.
```go
if err := one(); err != nil {
    if sdk.ErrorIs(err, sdk.ErrNotFound) {
        // do something specific for not found error
    }
}
```
