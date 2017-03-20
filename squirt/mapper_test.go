package squirt_test

import (
	"reflect"
	"testing"

	"github.com/sanity-io/go-squirt/squirt"
)

type simple struct {
	A string
	B int
}

func TestMapper_basic(t *testing.T) {
	squirt.Dump(squirt.MapReusedPointers(reflect.ValueOf([]string{"feh"})))

	str := &simple{"feh", 1}
	strB := &simple{"fneh", 2}
	squirt.Dump([]interface{}{str, str, strB, str})
	// squirt.Dump(squirt.MapReusedPointers(reflect.ValueOf([]interface{}{*str, str, strB})))
	// squirt.Dump(squirt.MapReusedPointers(reflect.ValueOf([]*simple{str, str, str, strB})))
}
