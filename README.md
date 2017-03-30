# Litter

Litter implements a deep pretty printer for Go data structures to aid in debugging. It is a limited
replacement for `go-spew` with focus on terseness in output to make life simpler when debugging complex
data structures. Its main reason for being is that it will detect circular references or aliasing and
replace additional references to the same object with aliases. Like this:


```go
type Circular struct {
  Self: *Circular
}

selfref := Circular{}
selfref.Self = &selfref

litter.Dump(selfref)
```

will output:

```go
Circular { // p0
  Self: p0,
}
```
(a small bonus: the output is valid Go fwiw)

## Installation

```bash
$ go get -u github.com/sanity-io/litter
```

## Quick Start

Add this import line to the file you're working in:

```go
import "github.com/sanity-io/litter"
```

To dump a variable with full newlines, indentation, type, and aliasing
information use Dump or Sdump:

```go
litter.Dump(myVar1)
str := litter.Sdump(myVar1)
```
## `litter.Dump(value)`
Dumps the data structure to STDOUT.

## `litter.Sdump(value)`
Returns the dump as a string

## Configuration
You can configure litter globally by modifying the default `litter.Config`

```go
litter.Config.StripPackageNames = true // strip all package names from types
litter.Config.HidePrivateFields = true // hide private struct fields from dumped structs
litter.Config.HomePackage = "mypackage" // sets a "home" pacage. The package name will be stripped from all its types
```
## `litter.Options`
Allows you to configure a local configuration of litter to allow for proper compartmentalization of state at the expense of some comfort:

``` go
  sq := litter.Options {
    HidePrivateFields: true,
    HomePackage: "thispack",
  })

  sq.Dump([]string("dumped", "with", "local", "settings"))
```

