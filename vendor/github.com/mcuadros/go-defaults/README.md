go-defaults [![Build Status](https://travis-ci.org/mcuadros/go-defaults.png?branch=master)](https://travis-ci.org/mcuadros/go-defaults) [![GoDoc](http://godoc.org/github.com/mcuadros/go-defaults?status.png)](http://godoc.org/github.com/mcuadros/go-defaults) [![GitHub release](https://img.shields.io/github/release/mcuadros/go-defaults.svg)](https://github.com/mcuadros/go-defaults/releases)
==============================

Enabling stuctures with defaults values using [struct tags](http://golang.org/pkg/reflect/#StructTag).

Installation
------------

The recommended way to install go-defaults

```
go get gopkg.in/mcuadros/go-defaults.v1
```

Examples
--------

A basic example:

```go
import (
    "fmt"
    "github.com/mcuadros/go-defaults"
)

type ExampleBasic struct {
    Foo bool   `default:"true"` //<-- StructTag with a default key
    Bar string `default:"33"`
    Qux int8
    Dur time.Duration `default:"1m"`
}

func NewExampleBasic() *ExampleBasic {
    example := new(ExampleBasic)
    defaults.SetDefaults(example) //<-- This set the defaults values

    return example
}

...

test := NewExampleBasic()
fmt.Println(test.Foo) //Prints: true
fmt.Println(test.Bar) //Prints: 33
fmt.Println(test.Qux) //Prints:
fmt.Println(test.Dur) //Prints: 1m0s
```

Caveats
-------

At the moment, the way the default filler checks whether it should fill a struct field or not is by comparing the current field value with the corresponding zero value of that type. This has a subtle implication: the zero value set explicitly by you will get overriden by default value during `SetDefaults()` call. So if you need to set the field to container zero value, you need to set it explicitly AFTER setting the defaults.

Take the basic example in the above section and change it slightly:
```go

example := ExampleBasic{
    Bar: 0,
}
defaults.SetDefaults(example)
fmt.Println(example.Bar) //Prints: 33 instead of 0 (which is zero value for int)

example.Bar = 0 // set needed zero value AFTER applying defaults
fmt.Println(example.Bar) //Prints: 0

```

License
-------

MIT, see [LICENSE](LICENSE)
