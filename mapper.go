package litter

import (
	"reflect"
	"sort"
)

// MapReusedPointers takes a structure, and recursively maps all pointers mentioned in the tree,
// detecting circular references, and providing a list of all pointers that was referenced at
// least twice by the provided structure.
func MapReusedPointers(v reflect.Value) []uintptr {
	pm := &pointerVisitor{}
	pm.consider(v)
	if len(pm.reusedPointers) == 0 {
		return nil
	}

	a := make([]uintptr, 0, len(pm.reusedPointers))
	for k := range pm.reusedPointers {
		a = append(a, k)
	}
	return a
}

type ptrmap map[uintptr]struct{}

func (pm *ptrmap) contains(p uintptr) bool {
	if *pm != nil {
		_, ok := (*pm)[p]
		return ok
	}
	return false
}

func (pm *ptrmap) add(p uintptr) {
	if !pm.contains(p) {
		if *pm == nil {
			*pm = make(map[uintptr]struct{}, 31)
		}
		(*pm)[p] = struct{}{}
	}
}

type pointerVisitor struct {
	pointers       ptrmap
	reusedPointers ptrmap
}

// Recursively consider v and each of its children, updating the map according to the
// semantics of MapReusedPointers
func (pm *pointerVisitor) consider(v reflect.Value) {
	if v.Kind() == reflect.Invalid {
		return
	}
	if isPointerValue(v) && v.Pointer() != 0 { // pointer is 0 for unexported fields
		if pm.addPointerReturnTrueIfWasReused(v.Pointer()) {
			// No use descending inside this value, since it have been seen before and all its descendants
			// have been considered
			return
		}
	}

	// Now descend into any children of this value
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		numEntries := v.Len()
		for i := 0; i < numEntries; i++ {
			pm.consider(v.Index(i))
		}

	case reflect.Interface:
		pm.consider(v.Elem())

	case reflect.Ptr:
		pm.consider(v.Elem())

	case reflect.Map:
		keys := v.MapKeys()
		sort.Sort(mapKeySorter{
			keys:    keys,
			options: &Config,
		})
		for _, key := range keys {
			pm.consider(v.MapIndex(key))
		}

	case reflect.Struct:
		numFields := v.NumField()
		for i := 0; i < numFields; i++ {
			pm.consider(v.Field(i))
		}
	}
}

// addPointer to the pointerMap, update reusedPointers. Returns true if pointer was reused
func (pm *pointerVisitor) addPointerReturnTrueIfWasReused(ptr uintptr) bool {
	// Is this allready known to be reused?
	if pm.reusedPointers.contains(ptr) {
		return true
	}

	// Have we seen it once before?
	if pm.pointers.contains(ptr) {
		// Add it to the register of pointers we have seen more than once
		pm.reusedPointers.add(ptr)
		return true
	}

	// This pointer was new to us
	pm.pointers.add(ptr)
	return false
}
