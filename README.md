# go-squirt

Go-squirt implements a deep pretty printer for Go data structures to aid in debugging. It is a limited
replacement for `go-spew` with focus on terseness in output to make life simpler when debugging complex
data structures. Its main reason for being is that it will detect circular references or aliasing and
replace additional references to the same object with aliases. Like this:


```go
type Circular struct {
  Self: *Circuler
}

selfref := Circular{}
selfred.Self = &selfref

squirt.Dump(selfref)
```

will output:

```
Circular { // p0
  Self: p0,
}
```


## Installation

```bash
$ go get -u github.com/sanity-io/go-squirt/squirt
```

## Quick Start

Add this import line to the file you're working in:

```go
import "github.com/sanity-io/go-squirt/squirt"
```

To dump a variable with full newlines, indentation, type, and aliasing
information use Dump or Sdump:

```go
squirt.Dump(myVar1)
str := squirt.Sdump(myVar1)
```
## `squirt.Dump(value)`
Dumps the data structure to STDOUT.

## `squirt.Sdump(value)`
Returns the dump as a string

## Configuration
You can configure squirt globally by modifying the default `squirt.Config`

```go
squirt.Config.StripPackageNames = true // strip all package names from types
squirt.Config.HidePrivateMembers = true // hide private members from dumped structs
squirt.Config.HomePackage = "mypackage" // sets a "home" pacage. The package name will be stripped from all its types
```
## `squirt.New(opts)`
Allows you to configure a local version of squirt to allow for proper compartmentalization of state at the
expense of some comfort:

``` go
  sq := squirt.New(squirt.Options {
    HidePrivateMembers: true,
    HomePackage: "thispack",
  })

  sq.Dump([]string("dumped", "with", "local", "settings"))
```

