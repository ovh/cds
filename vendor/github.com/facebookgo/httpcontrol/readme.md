httpcontrol [![Build Status](https://secure.travis-ci.org/facebookgo/httpcontrol.png)](http://travis-ci.org/facebookgo/httpcontrol)
===========

    import "github.com/facebookgo/httpcontrol"

Package httpcontrol allows a HTTP transport supporting connection pooling,
timeouts & retries.

This Transport is built on top of the standard library transport and augments it
with additional features. Using it can be as simple as:

    client := &http.Client{
        Transport: &httpcontrol.Transport{
            RequestTimeout: time.Minute,
            MaxTries: 3,
        },
    }
    res, err := client.Get("http://example.com/")

Documentation: http://godoc.org/github.com/facebookgo/httpcontrol
