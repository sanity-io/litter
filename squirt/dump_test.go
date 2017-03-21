package squirt_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"

	"github.com/sanity-io/go-squirt/squirt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type BlankStruct struct{}

type BasicStruct struct {
	Public  int
	private int
}

type InterfaceStruct struct {
	Ifc interface{}
}

type RecursiveStruct struct {
	Ptr *RecursiveStruct
}

var standardCfg = squirt.Options{}

func performDumpTestsWithCfg(t *testing.T, suiteName string, cfg *squirt.Options, cases interface{}) {
	referenceFileName := "testdata/" + suiteName + ".dump"
	dump := cfg.Sdump(cases)
	reference, err := ioutil.ReadFile(referenceFileName)
	if os.IsNotExist(err) {
		ioutil.WriteFile(referenceFileName, []byte(dump), 0644)
		return
	}
	AssertEqualStringsWithDiff(t, string(reference), dump)
}

func performDumpTests(t *testing.T, suiteName string, cases interface{}) {
	performDumpTestsWithCfg(t, suiteName, &standardCfg, cases)
}

func TestDump_primitives(t *testing.T) {
	performDumpTests(t, "primitives", []interface{}{
		false,
		true,
		7,
		int8(10),
		int16(10),
		int32(10),
		int64(10),
		uint8(10),
		uint16(10),
		uint32(10),
		uint64(10),
		uint(10),
		float32(12.3),
		float64(12.3),
		complex64(12 + 10.5i),
		complex128(-1.2 - 0.1i),
		"string with \"quote\"",
		[]int{1, 2, 3},
		interface{}("hello from interface"),
		BlankStruct{},
		&BlankStruct{},
		BasicStruct{1, 2},
		nil,
		interface{}(nil),
	})
}

func TestDump_pointerAliasing(t *testing.T) {
	p0 := &RecursiveStruct{Ptr: nil}
	p1 := &RecursiveStruct{Ptr: p0}
	p2 := &RecursiveStruct{}
	p2.Ptr = p2

	performDumpTests(t, "pointerAliasing", []*RecursiveStruct{
		p0,
		p0,
		p1,
		p2,
	})
}

func TestDump_nilIntefacesInStructs(t *testing.T) {
	p0 := &InterfaceStruct{nil}
	p1 := &InterfaceStruct{p0}

	performDumpTests(t, "nilIntefacesInStructs", []*InterfaceStruct{
		p0,
		p1,
		p0,
		nil,
	})
}

func TestDump_config(t *testing.T) {
	data := []interface{}{
		squirt.Config,
		&BasicStruct{1, 2},
	}
	performDumpTestsWithCfg(t, "config_HidePrivateFields", &squirt.Options{
		HidePrivateFields: true,
	}, data)
	performDumpTestsWithCfg(t, "config_StripPackageNames", &squirt.Options{
		StripPackageNames: true,
	}, data)
	performDumpTestsWithCfg(t, "config_HomePackage", &squirt.Options{
		HomePackage: "squirt_test",
	}, data)
}

func TestDump_maps(t *testing.T) {
	performDumpTests(t, "maps", []interface{}{
		map[string]string{
			"hello": "there",
		},
		map[int]string{
			1: "one",
			2: "two",
		},
		map[int]*BlankStruct{
			2: &BlankStruct{},
		},
	})
}

func DiffStrings(t *testing.T, expected, actual string) (*string, bool) {
	if actual == expected {
		return nil, true
	}

	dir, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	require.NoError(t, ioutil.WriteFile(fmt.Sprintf("%s/expected", dir), []byte(expected), 0644))
	require.NoError(t, ioutil.WriteFile(fmt.Sprintf("%s/actual", dir), []byte(actual), 0644))

	out, err := exec.Command("diff", "--side-by-side",
		fmt.Sprintf("%s/expected", dir),
		fmt.Sprintf("%s/actual", dir)).Output()
	if _, ok := err.(*exec.ExitError); !ok {
		require.NoError(t, err)
	}

	diff := string(out)
	return &diff, false
}

func AssertEqualStringsWithDiff(t *testing.T, expected, actual string,
	msgAndArgs ...interface{}) bool {
	diff, ok := DiffStrings(t, expected, actual)
	if ok {
		return true
	}

	message := messageFromMsgAndArgs(msgAndArgs...)
	if message == "" {
		message = "Strings are different"
	}
	assert.Fail(t, fmt.Sprintf("%s (left is expected, right is actual):\n%s", message, *diff))
	return false
}

func messageFromMsgAndArgs(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 || msgAndArgs == nil {
		return ""
	}
	if len(msgAndArgs) == 1 {
		return msgAndArgs[0].(string)
	}
	if len(msgAndArgs) > 1 {
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
	return ""
}
