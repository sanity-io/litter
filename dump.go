package litter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	packageNameStripperRegexp = regexp.MustCompile(`\b[a-zA-Z_]+[a-zA-Z_0-9]+\.`)
	compactTypeRegexp         = regexp.MustCompile(`\s*([,;{}()])\s*`)
)

// Dumper is the interface for implementing custom dumper for your types.
type Dumper interface {
	LitterDump(w io.Writer)
}

// Options represents configuration options for litter
type Options struct {
	Compact           bool
	StripPackageNames bool
	HidePrivateFields bool
	HideZeroValues    bool
	FieldExclusions   *regexp.Regexp
	FieldFilter       func(reflect.StructField, reflect.Value) bool
	HomePackage       string
	Separator         string
	StrictGo          bool
	DumpFunc          func(reflect.Value, io.Writer) bool

	// DisablePointerReplacement, if true, disables the replacing of pointer data with variable names
	// when it's safe. This is useful for diffing two structures, where pointer variables would cause
	// false changes. However, circular graphs are still detected and elided to avoid infinite output.
	DisablePointerReplacement bool

	// FormatTime, if true, will format [time.Time] values.
	FormatTime bool
}

// Config is the default config used when calling Dump
var Config = Options{
	StripPackageNames: false,
	HidePrivateFields: true,
	FieldExclusions:   regexp.MustCompile(`^(XXX_.*)$`), // XXX_ is a prefix of fields generated by protoc-gen-go
	Separator:         " ",
}

type dumpState struct {
	w                 io.Writer
	depth             int
	config            *Options
	pointers          ptrmap
	visitedPointers   ptrmap
	parentPointers    ptrmap
	currentPointer    *ptrinfo
	homePackageRegexp *regexp.Regexp
	timeFormatter     func(t time.Time) string
}

func (s *dumpState) write(b []byte) {
	if _, err := s.w.Write(b); err != nil {
		panic(err)
	}
}

func (s *dumpState) writeString(str string) {
	s.write([]byte(str))
}

func (s *dumpState) indent() {
	if !s.config.Compact {
		s.write(bytes.Repeat([]byte("  "), s.depth))
	}
}

func (s *dumpState) newlineWithPointerNameComment() {
	if ptr := s.currentPointer; ptr != nil {
		if s.config.Compact {
			s.write([]byte(fmt.Sprintf("/*%s*/", ptr.label())))
		} else {
			s.write([]byte(fmt.Sprintf(" // %s\n", ptr.label())))
		}
		s.currentPointer = nil
		return
	}
	if !s.config.Compact {
		s.write([]byte("\n"))
	}
}

func (s *dumpState) dumpType(v reflect.Value) {
	typeName := v.Type().String()
	if s.config.StripPackageNames {
		typeName = packageNameStripperRegexp.ReplaceAllLiteralString(typeName, "")
	} else if s.homePackageRegexp != nil {
		typeName = s.homePackageRegexp.ReplaceAllLiteralString(typeName, "")
	}
	if s.config.Compact {
		typeName = compactTypeRegexp.ReplaceAllString(typeName, "$1")
	}
	s.write([]byte(typeName))
}

func (s *dumpState) dumpSlice(v reflect.Value) {
	s.dumpType(v)
	numEntries := v.Len()
	if numEntries == 0 {
		s.write([]byte("{}"))
		return
	}
	s.write([]byte("{"))
	s.newlineWithPointerNameComment()
	s.depth++
	for i := 0; i < numEntries; i++ {
		s.indent()
		s.dumpVal(v.Index(i))
		if !s.config.Compact || i < numEntries-1 {
			s.write([]byte(","))
		}
		s.newlineWithPointerNameComment()
	}
	s.depth--
	s.indent()
	s.write([]byte("}"))
}

func (s *dumpState) dumpStruct(v reflect.Value) {
	val := v.Interface()
	if t, ok := val.(time.Time); ok && s.timeFormatter != nil {
		s.writeString(s.timeFormatter(t))
		return
	}

	dumpPreamble := func() {
		s.dumpType(v)
		s.write([]byte("{"))
		s.newlineWithPointerNameComment()
		s.depth++
	}
	preambleDumped := false
	vt := v.Type()
	numFields := v.NumField()
	for i := 0; i < numFields; i++ {
		vtf := vt.Field(i)
		if s.config.HidePrivateFields && vtf.PkgPath != "" || s.config.FieldExclusions != nil && s.config.FieldExclusions.MatchString(vtf.Name) {
			continue
		}
		if s.config.FieldFilter != nil && !s.config.FieldFilter(vtf, v.Field(i)) {
			continue
		}
		if s.config.HideZeroValues && isZeroValue(v.Field(i)) {
			continue
		}
		if !preambleDumped {
			dumpPreamble()
			preambleDumped = true
		}
		s.indent()
		s.write([]byte(vtf.Name))
		if s.config.Compact {
			s.write([]byte(":"))
		} else {
			s.write([]byte(": "))
		}
		s.dumpVal(v.Field(i))
		if !s.config.Compact || i < numFields-1 {
			s.write([]byte(","))
		}
		s.newlineWithPointerNameComment()
	}
	if preambleDumped {
		s.depth--
		s.indent()
		s.write([]byte("}"))
	} else {
		// There were no fields dumped
		s.dumpType(v)
		s.write([]byte("{}"))
	}
}

func (s *dumpState) dumpMap(v reflect.Value) {
	if v.IsNil() {
		s.dumpType(v)
		s.writeString("(nil)")
		return
	}

	s.dumpType(v)

	keys := v.MapKeys()
	if len(keys) == 0 {
		s.write([]byte("{}"))
		return
	}

	s.write([]byte("{"))
	s.newlineWithPointerNameComment()
	s.depth++
	sort.Sort(mapKeySorter{
		keys:    keys,
		options: s.config,
	})
	numKeys := len(keys)
	for i, key := range keys {
		s.indent()
		s.dumpVal(key)
		if s.config.Compact {
			s.write([]byte(":"))
		} else {
			s.write([]byte(": "))
		}
		s.dumpVal(v.MapIndex(key))
		if !s.config.Compact || i < numKeys-1 {
			s.write([]byte(","))
		}
		s.newlineWithPointerNameComment()
	}
	s.depth--
	s.indent()
	s.write([]byte("}"))
}

func (s *dumpState) dumpFunc(v reflect.Value) {
	parts := strings.Split(runtime.FuncForPC(v.Pointer()).Name(), "/")
	name := parts[len(parts)-1]

	// Anonymous function
	if strings.Count(name, ".") > 1 {
		s.dumpType(v)
	} else {
		if s.config.StripPackageNames {
			name = packageNameStripperRegexp.ReplaceAllLiteralString(name, "")
		} else if s.homePackageRegexp != nil {
			name = s.homePackageRegexp.ReplaceAllLiteralString(name, "")
		}
		if s.config.Compact {
			name = compactTypeRegexp.ReplaceAllString(name, "$1")
		}
		s.write([]byte(name))
	}
}

func (s *dumpState) dumpChan(v reflect.Value) {
	vType := v.Type()
	res := []byte(vType.String())
	s.write(res)
}

func (s *dumpState) dumpCustom(v reflect.Value, buf *bytes.Buffer) {
	// Dump the type
	s.dumpType(v)

	if s.config.Compact {
		s.write(buf.Bytes())
		return
	}

	// Now output the dump taking care to apply the current indentation-level
	// and pointer name comments.
	var err error
	firstLine := true
	for err == nil {
		var lineBytes []byte
		lineBytes, err = buf.ReadBytes('\n')
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
		s.write([]byte(line))

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

var dumperType = reflect.TypeOf((*Dumper)(nil)).Elem()

func (s *dumpState) descendIntoPossiblePointer(value reflect.Value, f func()) {
	canonicalize := true
	if isPointerValue(value) {
		// If elision disabled, and this is not a circular reference, don't canonicalize
		if s.config.DisablePointerReplacement && s.parentPointers.add(value) {
			canonicalize = false
		}

		// Add to stack of pointers we're recursively descending into
		s.parentPointers.add(value)
		defer s.parentPointers.remove(value)
	}

	if !canonicalize {
		ptr, _ := s.pointerFor(value)
		s.currentPointer = ptr
		f()
		return
	}

	ptr, firstVisit := s.pointerFor(value)
	if ptr == nil {
		f()
		return
	}
	if firstVisit {
		s.currentPointer = ptr
		f()
		return
	}
	s.write([]byte(ptr.label()))
}

func (s *dumpState) dumpVal(value reflect.Value) {
	if value.Kind() == reflect.Ptr && value.IsNil() {
		s.write([]byte("nil"))
		return
	}

	v := deInterface(value)
	kind := v.Kind()

	// Try to handle with dump func
	if s.config.DumpFunc != nil {
		buf := new(bytes.Buffer)
		if s.config.DumpFunc(v, buf) {
			s.dumpCustom(v, buf)
			return
		}
	}

	// Handle custom dumpers
	if v.Type().Implements(dumperType) {
		s.descendIntoPossiblePointer(v, func() {
			// Run the custom dumper buffering the output
			buf := new(bytes.Buffer)
			dumpFunc := v.MethodByName("LitterDump")
			dumpFunc.Call([]reflect.Value{reflect.ValueOf(buf)})
			s.dumpCustom(v, buf)
		})
		return
	}

	switch kind {
	case reflect.Invalid:
		// Do nothing.  We should never get here since invalid has already
		// been handled above.
		s.write([]byte("<invalid>"))

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
		s.write([]byte(strconv.Quote(v.String())))

	case reflect.Slice:
		if v.IsNil() {
			printNil(s.w)
			break
		}
		fallthrough

	case reflect.Array:
		s.descendIntoPossiblePointer(v, func() {
			s.dumpSlice(v)
		})

	case reflect.Interface:
		// The only time we should get here is for nil interfaces due to
		// unpackValue calls.
		if v.IsNil() {
			printNil(s.w)
		}

	case reflect.Ptr:
		s.descendIntoPossiblePointer(v, func() {
			if s.config.StrictGo {
				s.writeString(fmt.Sprintf("(func(v %s) *%s { return &v })(", v.Elem().Type(), v.Elem().Type()))
				s.dumpVal(v.Elem())
				s.writeString(")")
			} else {
				s.writeString("&")
				s.dumpVal(v.Elem())
			}
		})

	case reflect.Map:
		s.descendIntoPossiblePointer(v, func() {
			s.dumpMap(v)
		})

	case reflect.Struct:
		s.dumpStruct(v)

	case reflect.Func:
		s.dumpFunc(v)

	case reflect.Chan:
		s.dumpChan(v)

	default:
		if v.CanInterface() {
			s.writeString(fmt.Sprintf("%v", v.Interface()))
		} else {
			s.writeString(fmt.Sprintf("%v", v.String()))
		}
	}
}

// registers that the value has been visited and checks to see if it is one of the
// pointers we will see multiple times. If it is, it returns a temporary name for this
// pointer. It also returns a boolean value indicating whether this is the first time
// this name is returned so the caller can decide whether the contents of the pointer
// has been dumped before or not.
func (s *dumpState) pointerFor(v reflect.Value) (*ptrinfo, bool) {
	if isPointerValue(v) {
		if info, ok := s.pointers.get(v); ok {
			firstVisit := s.visitedPointers.add(v)
			return info, firstVisit
		}
	}
	return nil, false
}

// prepares a new state object for dumping the provided value
func newDumpState(value reflect.Value, options *Options, writer io.Writer) *dumpState {
	result := &dumpState{
		config:   options,
		pointers: mapReusedPointers(value),
		w:        writer,
	}

	if options.FormatTime {
		result.timeFormatter = func(t time.Time) string {
			t = t.In(time.UTC)
			return fmt.Sprintf(
				`time.Date(%d, %d, %d, %d, %d, %d, %d, time.UTC)`,
				t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(),
			)
		}
	}

	if options.HomePackage != "" {
		result.homePackageRegexp = regexp.MustCompile(fmt.Sprintf("\\b%s\\.", options.HomePackage))
	}

	return result
}

// Dump a value to stdout.
func Dump(value ...interface{}) {
	(&Config).Dump(value...)
}

// D dumps a value to stdout, and is a shorthand for [Dump].
func D(value ...interface{}) {
	Dump(value...)
}

// Sdump dumps a value to a string.
func Sdump(value ...interface{}) string {
	return (&Config).Sdump(value...)
}

// Dump a value to stdout according to the options
func (o Options) Dump(values ...interface{}) {
	for i, value := range values {
		state := newDumpState(reflect.ValueOf(value), &o, os.Stdout)
		if i > 0 {
			state.write([]byte(o.Separator))
		}
		state.dump(value)
	}
	_, _ = os.Stdout.Write([]byte("\n"))
}

// Sdump dumps a value to a string according to the options
func (o Options) Sdump(values ...interface{}) string {
	buf := new(bytes.Buffer)
	for i, value := range values {
		if i > 0 {
			_, _ = buf.Write([]byte(o.Separator))
		}
		state := newDumpState(reflect.ValueOf(value), &o, buf)
		state.dump(value)
	}
	return buf.String()
}

type mapKeySorter struct {
	keys    []reflect.Value
	options *Options
}

func (s mapKeySorter) Len() int {
	return len(s.keys)
}

func (s mapKeySorter) Swap(i, j int) {
	s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
}

func (s mapKeySorter) Less(i, j int) bool {
	ibuf := new(bytes.Buffer)
	jbuf := new(bytes.Buffer)
	newDumpState(s.keys[i], s.options, ibuf).dumpVal(s.keys[i])
	newDumpState(s.keys[j], s.options, jbuf).dumpVal(s.keys[j])
	return ibuf.String() < jbuf.String()
}
