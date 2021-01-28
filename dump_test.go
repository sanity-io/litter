package litter_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanity-io/litter"
)

func Function(arg1 string, arg2 int) (string, error) {
	return "", nil
}

type BlankStruct struct{}

type BasicStruct struct {
	Public  int
	private int
}

type IntAlias int

type InterfaceStruct struct {
	Ifc interface{}
}

type RecursiveStruct struct {
	Ptr *RecursiveStruct
}

type CustomMultiLineDumper struct {
	Dummy int
}

func (cmld *CustomMultiLineDumper) LitterDump(w io.Writer) {
	_, _ = w.Write([]byte("{\n  multi\n  line\n}"))
}

type CustomSingleLineDumper int

func (csld CustomSingleLineDumper) LitterDump(w io.Writer) {
	_, _ = w.Write([]byte("<custom>"))
}

func TestSdump_primitives(t *testing.T) {
	runTests(t, "primitives", []interface{}{
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
		(func(v int) *int { return &v })(10),
		"string with \"quote\"",
		[]int{1, 2, 3},
		interface{}("hello from interface"),
		BlankStruct{},
		&BlankStruct{},
		BasicStruct{1, 2},
		IntAlias(10),
		(func(v IntAlias) *IntAlias { return &v })(10),
		Function,
		func(arg string) (bool, error) { return false, nil },
		nil,
		interface{}(nil),
	})
}

func TestSdump_customDumper(t *testing.T) {
	cmld := CustomMultiLineDumper{Dummy: 1}
	cmld2 := CustomMultiLineDumper{Dummy: 2}
	csld := CustomSingleLineDumper(42)
	csld2 := CustomSingleLineDumper(43)
	runTests(t, "customDumper", map[string]interface{}{
		"v1":  &cmld,
		"v2":  &cmld,
		"v2x": &cmld2,
		"v3":  csld,
		"v4":  &csld,
		"v5":  &csld,
		"v6":  &csld2,
	})
}

func TestSdump_pointerAliasing(t *testing.T) {
	p0 := &RecursiveStruct{Ptr: nil}
	p1 := &RecursiveStruct{Ptr: p0}
	p2 := &RecursiveStruct{}
	p2.Ptr = p2

	runTests(t, "pointerAliasing", []*RecursiveStruct{
		p0,
		p0,
		p1,
		p2,
	})
}

func TestSdump_nilIntefacesInStructs(t *testing.T) {
	p0 := &InterfaceStruct{nil}
	p1 := &InterfaceStruct{p0}

	runTests(t, "nilIntefacesInStructs", []*InterfaceStruct{
		p0,
		p1,
		p0,
		nil,
	})
}

func TestSdump_config(t *testing.T) {
	type options struct {
		Compact           bool
		StripPackageNames bool
		HidePrivateFields bool
		HomePackage       string
		Separator         string
		StrictGo          bool
	}

	opts := options{
		StripPackageNames: false,
		HidePrivateFields: true,
		Separator:         " ",
	}

	data := []interface{}{
		opts,
		&BasicStruct{1, 2},
		Function,
		(func(v int) *int { return &v })(20),
		(func(v IntAlias) *IntAlias { return &v })(20),
		litter.Dump,
		func(s string, i int) (bool, error) { return false, nil },
	}

	runTestWithCfg(t, "config_Compact", &litter.Options{
		Compact: true,
	}, data)
	runTestWithCfg(t, "config_HidePrivateFields", &litter.Options{
		HidePrivateFields: true,
	}, data)
	runTestWithCfg(t, "config_HideZeroValues", &litter.Options{
		HideZeroValues: true,
	}, data)
	runTestWithCfg(t, "config_StripPackageNames", &litter.Options{
		StripPackageNames: true,
	}, data)
	runTestWithCfg(t, "config_HomePackage", &litter.Options{
		HomePackage: "litter_test",
	}, data)
	runTestWithCfg(t, "config_FieldFilter", &litter.Options{
		FieldFilter: func(f reflect.StructField, v reflect.Value) bool {
			return f.Type.Kind() == reflect.String
		},
	}, data)
	runTestWithCfg(t, "config_StrictGo", &litter.Options{
		StrictGo: true,
	}, data)
	runTestWithCfg(t, "config_DumpFunc", &litter.Options{
		DumpFunc: func(v reflect.Value, w io.Writer) bool {
			if !v.CanInterface() {
				return false
			}
			if b, ok := v.Interface().(bool); ok {
				if b {
					io.WriteString(w, `"on"`)
				} else {
					io.WriteString(w, `"off"`)
				}
				return true
			}
			return false
		},
	}, data)

	basic := &BasicStruct{1, 2}
	runTestWithCfg(t, "config_DisablePointerReplacement_simpleReusedStruct", &litter.Options{
		DisablePointerReplacement: true,
	}, []interface{}{basic, basic})
	circular := &RecursiveStruct{}
	circular.Ptr = circular
	runTestWithCfg(t, "config_DisablePointerReplacement_circular", &litter.Options{
		DisablePointerReplacement: true,
	}, circular)
}

func TestSdump_multipleArgs(t *testing.T) {
	value1 := []string{"x", "y"}
	value2 := int32(42)

	runTestWithCfg(t, "multipleArgs_noSeparator", &litter.Options{}, value1, value2)
	runTestWithCfg(t, "multipleArgs_lineBreak", &litter.Options{Separator: "\n"}, value1, value2)
	runTestWithCfg(t, "multipleArgs_separator", &litter.Options{Separator: "***"}, value1, value2)
}

func TestSdump_maps(t *testing.T) {
	runTests(t, "maps", []interface{}{
		map[string]string{
			"hello":          "there",
			"something":      "something something",
			"another string": "indeed",
		},
		map[int]string{
			3: "three",
			1: "one",
			2: "two",
		},
		map[int]*BlankStruct{
			2: &BlankStruct{},
		},
	})
}

var standardCfg = litter.Options{}

func runTestWithCfg(t *testing.T, name string, cfg *litter.Options, cases ...interface{}) {
	t.Run(name, func(t *testing.T) {
		fileName := fmt.Sprintf("testdata/%s.dump", name)
		dump := cfg.Sdump(cases...)
		reference, err := ioutil.ReadFile(fileName)
		if os.IsNotExist(err) {
			t.Logf("Note: Test data file %s does not exist, writing it; verify contents!", fileName)
			err := ioutil.WriteFile(fileName, []byte(dump), 0644)
			if err != nil {
				t.Error(err)
			}
			return
		}
		assertEqualStringsWithDiff(t, string(reference), dump)
	})
}

func runTests(t *testing.T, name string, cases ...interface{}) {
	runTestWithCfg(t, name, &standardCfg, cases...)
}

func diffStrings(t *testing.T, expected, actual string) (*string, bool) {
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

func assertEqualStringsWithDiff(t *testing.T, expected, actual string,
	msgAndArgs ...interface{}) bool {
	diff, ok := diffStrings(t, expected, actual)
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
