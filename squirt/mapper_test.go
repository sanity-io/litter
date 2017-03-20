package squirt_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/sanity-io/go-squirt/squirt"
)

type simple struct {
	A string
	b int
}

func TestMapper_basic(t *testing.T) {
	squirt.Dump(squirt.MapReusedPointers(reflect.ValueOf([]string{"feh"})))

	str := &simple{"feh", 1}
	strB := &simple{"fneh", 2}
	now := time.Now()
	squirt.New(squirt.Options{
		HomePackage:        "squirt_test",
		HidePrivateMembers: true,
	}).Dump([]interface{}{str, str, strB, str, &now, &now})
	// squirt.Dump(squirt.MapReusedPointers(reflect.ValueOf([]interface{}{*str, str, strB})))
	// squirt.Dump(squirt.MapReusedPointers(reflect.ValueOf([]*simple{str, str, str, strB})))
}
