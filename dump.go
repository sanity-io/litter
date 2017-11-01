package litter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var packageNameStripperRegexp = regexp.MustCompile("\\b[a-zA-Z_]+[a-zA-Z_0-9]+\\.")

// Dumper is the interface for implementing custom dumper for your types.
type Dumper interface {
	LitterDump(w io.Writer)
}

// Options represents configuration options for litter
type Options struct {
	StripPackageNames bool
	HidePrivateFields bool
	HomePackage       string
	Separator         string
}

// Config is the default config used when calling Dump
var Config = Options{
	StripPackageNames: false,
	HidePrivateFields: true,
	Separator:         " ",
}

type dumpState struct {
	w                  io.Writer
	depth              int
	config             *Options
	pointers           []uintptr
	visitedPointers    []uintptr
	currentPointerName string
	homePackageRegexp  *regexp.Regexp
}

func (s *dumpState) indent() {
	s.w.Write(bytes.Repeat([]byte("  "), s.depth))
}

func (s *dumpState) newlineWithPointerNameComment() {
	if s.currentPointerName != "" {
		s.w.Write([]byte(fmt.Sprintf(" // %s\n", s.currentPointerName)))
		s.currentPointerName = ""
		return
	}
	s.w.Write([]byte("\n"))
}

func (s *dumpState) dumpType(v reflect.Value) {
	typeName := v.Type().String()
	if s.config.StripPackageNames {
		typeName = packageNameStripperRegexp.ReplaceAllLiteralString(typeName, "")
	} else if s.homePackageRegexp != nil {
		typeName = s.homePackageRegexp.ReplaceAllLiteralString(typeName, "")
	}
	s.w.Write([]byte(typeName))
}

func (s *dumpState) dumpSlice(v reflect.Value) {
	s.dumpType(v)
	numEntries := v.Len()
	if numEntries == 0 {
		s.w.Write([]byte("{}"))
		s.newlineWithPointerNameComment()
		return
	}
	s.w.Write([]byte("{"))
	s.newlineWithPointerNameComment()
	s.depth++
	for i := 0; i < numEntries; i++ {
		s.indent()
		s.dumpVal(v.Index(i))
		s.w.Write([]byte(","))
		s.newlineWithPointerNameComment()
	}
	s.depth--
	s.indent()
	s.w.Write([]byte("}"))
}

func (s *dumpState) dumpStruct(v reflect.Value) {
	dumpPreamble := func() {
		s.dumpType(v)
		s.w.Write([]byte("{"))
		s.newlineWithPointerNameComment()
		s.depth++
	}
	preambleDumped := false
	vt := v.Type()
	numFields := v.NumField()
	for i := 0; i < numFields; i++ {
		vtf := vt.Field(i)
		if s.config.HidePrivateFields && vtf.PkgPath != "" {
			continue
		}
		if !preambleDumped {
			dumpPreamble()
			preambleDumped = true
		}
		s.indent()
		s.w.Write([]byte(vtf.Name))
		s.w.Write([]byte(": "))
		s.dumpVal(v.Field(i))
		s.w.Write([]byte(","))
		s.newlineWithPointerNameComment()
	}
	if preambleDumped {
		s.depth--
		s.indent()
		s.w.Write([]byte("}"))
	} else {
		// There were no fields dumped
		s.dumpType(v)
		s.w.Write([]byte("{}"))
	}
}

func (s *dumpState) dumpMap(v reflect.Value) {
	s.dumpType(v)
	s.w.Write([]byte("{"))
	s.newlineWithPointerNameComment()
	s.depth++
	keys := v.MapKeys()
	sort.Sort(mapKeySorter{keys})
	for _, key := range keys {
		s.indent()
		s.dumpVal(key)
		s.w.Write([]byte(": "))
		s.dumpVal(v.MapIndex(key))
		s.w.Write([]byte(","))
		s.newlineWithPointerNameComment()
	}
	s.depth--
	s.indent()
	s.w.Write([]byte("}"))
}

func (s *dumpState) dumpCustom(v reflect.Value) {
	// Run the custom dumper buffering the output
	buf := new(bytes.Buffer)
	dumpFunc := v.MethodByName("LitterDump")
	dumpFunc.Call([]reflect.Value{reflect.ValueOf(buf)})

	// Dump the type
	s.dumpType(v)

	// Now output the dump taking care to apply the current indentation-level
	// and pointer name comments.
	var err error
	firstLine := true
	for err == nil {
		lineBytes, err := buf.ReadBytes('\n')
		line := strings.TrimRight(string(lineBytes), " \n")

		if err != nil && err != io.EOF {
			break
		}
		// Do not indent first line
		if firstLine {
			firstLine = false
		} else {
			s.indent()
		}
		s.w.Write([]byte(line))
		// At EOF we're done
		if err == io.EOF {
			return
		}
		s.newlineWithPointerNameComment()
	}
	panic(err)
}

func (s *dumpState) dump(value interface{}) {
	if value == nil {
		printNil(s.w)
		return
	}
	v := reflect.ValueOf(value)
	s.dumpVal(v)
}

func (s *dumpState) handlePointerAliasingAndCheckIfShouldDescend(value reflect.Value) bool {
	pointerName, firstVisit := s.pointerNameFor(value)
	if pointerName == "" {
		return true
	}
	if firstVisit {
		s.currentPointerName = pointerName
		return true
	}
	s.w.Write([]byte(pointerName))
	return false
}

func (s *dumpState) dumpVal(value reflect.Value) {
	if value.Kind() == reflect.Ptr && value.IsNil() {
		s.w.Write([]byte("nil"))
		return
	}

	v := deInterface(value)
	kind := v.Kind()

	// Handle custom dumpers
	dumperType := reflect.TypeOf((*Dumper)(nil)).Elem()
	if v.Type().Implements(dumperType) {
		if s.handlePointerAliasingAndCheckIfShouldDescend(v) {
			s.dumpCustom(v)
		}
		return
	}

	switch kind {
	case reflect.Invalid:
		// Do nothing.  We should never get here since invalid has already
		// been handled above.
		s.w.Write([]byte("<invalid>"))

	case reflect.Bool:
		printBool(s.w, v.Bool())

	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		printInt(s.w, v.Int(), 10)

	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		printUint(s.w, v.Uint(), 10)

	case reflect.Float32:
		printFloat(s.w, v.Float(), 32)

	case reflect.Float64:
		printFloat(s.w, v.Float(), 64)

	case reflect.Complex64:
		printComplex(s.w, v.Complex(), 32)

	case reflect.Complex128:
		printComplex(s.w, v.Complex(), 64)

	case reflect.String:
		s.w.Write([]byte(strconv.Quote(v.String())))

	case reflect.Slice:
		if v.IsNil() {
			printNil(s.w)
			break
		}
		fallthrough

	case reflect.Array:
		if s.handlePointerAliasingAndCheckIfShouldDescend(v) {
			s.dumpSlice(v)
		}

	case reflect.Interface:
		// The only time we should get here is for nil interfaces due to
		// unpackValue calls.
		if v.IsNil() {
			printNil(s.w)
		}

	case reflect.Ptr:
		if s.handlePointerAliasingAndCheckIfShouldDescend(v) {
			s.w.Write([]byte("&"))
			s.dumpVal(v.Elem())
		}

	case reflect.Map:
		if s.handlePointerAliasingAndCheckIfShouldDescend(v) {
			s.dumpMap(v)
		}

	case reflect.Struct:
		s.dumpStruct(v)

	default:
		if v.CanInterface() {
			fmt.Fprintf(s.w, "%v", v.Interface())
		} else {
			fmt.Fprintf(s.w, "%v", v.String())
		}
	}
}

// call to signal that the pointer is being visited, returns true if this is the
// first visit to that pointer. Used to detect when to output the entire contents
// behind a pointer (the first time), and when to just emit a name (all other times)
func (s *dumpState) visitPointerAndCheckIfItIsTheFirstTime(ptr uintptr) bool {
	for _, addr := range s.visitedPointers {
		if addr == ptr {
			return false
		}
	}
	s.visitedPointers = append(s.visitedPointers, ptr)
	return true
}

// registers that the value has been visited and checks to see if it is one of the
// pointers we will see multiple times. If it is, it returns a temporary name for this
// pointer. It also returns a boolean value indicating whether this is the first time
// this name is returned so the caller can decide whether the contents of the pointer
// has been dumped before or not.
func (s *dumpState) pointerNameFor(v reflect.Value) (string, bool) {
	if isPointerValue(v) {
		ptr := v.Pointer()
		for i, addr := range s.pointers {
			if ptr == addr {
				firstVisit := s.visitPointerAndCheckIfItIsTheFirstTime(ptr)
				return fmt.Sprintf("p%d", i), firstVisit
			}
		}
	}
	return "", false
}

// prepares a new state object for dumping the provided value
func newDumpState(value interface{}, options *Options) *dumpState {
	result := &dumpState{
		config:   options,
		pointers: MapReusedPointers(reflect.ValueOf(value)),
	}

	if options.HomePackage != "" {
		result.homePackageRegexp = regexp.MustCompile(fmt.Sprintf("\\b%s\\.", options.HomePackage))
	}

	return result
}

// Dump a value to stdout
func Dump(value interface{}) {
	(&Config).Dump(value)
}

// Sdump dumps a value to a string
func Sdump(value interface{}) string {
	return (&Config).Sdump(value)
}

// Dump a value to stdout according to the options
func (o Options) Dump(values ...interface{}) {
	for i, value := range values {
		state := newDumpState(value, &o)
		state.w = os.Stdout
		if i > 0 {
			state.w.Write([]byte(o.Separator))
		}
		state.dump(value)
	}
	os.Stdout.Write([]byte("\n"))
}

// Sdump dumps a value to a string according to the options
func (o Options) Sdump(values ...interface{}) string {
	buf := new(bytes.Buffer)
	for i, value := range values {
		if i > 0 {
			buf.Write([]byte(o.Separator))
		}
		state := newDumpState(value, &o)
		state.w = buf
		state.dump(value)
	}
	return buf.String()
}

type mapKeySorter struct {
	keys []reflect.Value
}

func (s mapKeySorter) Len() int {
	return len(s.keys)
}

func (s mapKeySorter) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}

func (s mapKeySorter) Less(i, j int) bool {
	return fmt.Sprintf("%s", s.keys[i].Interface()) < fmt.Sprintf("%s", s.keys[j].Interface())
}
