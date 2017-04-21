package litter

import "reflect"

type pointerMap struct {
	pointers       []uintptr
	reusedPointers []uintptr
}

// MapReusedPointers : Given a structure, it will recurively map all pointers mentioned in the tree, breaking
// circular references and provide a list of all pointers that was referenced at least twice
// by the provided structure.
func MapReusedPointers(v reflect.Value) []uintptr {
	pm := &pointerMap{
		reusedPointers: []uintptr{},
	}
	pm.consider(v)
	return pm.reusedPointers
}

// Recursively consider v and each of its children, updating the map according to the
// semantics of MapReusedPointers
func (pm *pointerMap) consider(v reflect.Value) {
	if v.Kind() == reflect.Invalid {
		return
	}
	// fmt.Printf("Considering [%s] %#v\n\r", v.Type().String(), v.Interface())
	if isPointerValue(v) && v.Pointer() != 0 { // pointer is 0 for unexported fields
		// fmt.Printf("Ptr is %d\n\r", v.Pointer())
		reused := pm.addPointerReturnTrueIfWasReused(v.Pointer())
		if reused {
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
		for _, key := range v.MapKeys() {
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
func (pm *pointerMap) addPointerReturnTrueIfWasReused(ptr uintptr) bool {
	// Is this allready known to be reused?
	for _, have := range pm.reusedPointers {
		if ptr == have {
			return true
		}
	}
	// Have we seen it once before?
	for _, seen := range pm.pointers {
		if ptr == seen {
			// Add it to the register of pointers we have seen more than once
			pm.reusedPointers = append(pm.reusedPointers, ptr)
			return true
		}
	}
	// This pointer was new to us
	pm.pointers = append(pm.pointers, ptr)
	return false
}
