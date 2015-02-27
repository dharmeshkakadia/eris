package types

import (
	"github.com/fatih/structs"
	"reflect"
)

// Values that are exposed to otto must be of a certain kind. They should only consist of basic types such as
// primitives, arrays/slices, and maps. This goes for values that are returned by functions as well. This function
// takes a given go value and turns it into an Atë compatible value.
//
// Generally speaking, numbers, strings and booleans are passed right in. Structs (and pointers to structs)
// are transformed into map[string]interface{} objects. This is done recursively. Arrays and slices are converted
// into arrays of interfaces. If the elements are structs, then they are converted to maps, and a new array is
// created.
//
// Primitive here means anything that can be converted directly into a javascript string, number or boolean. Everything else 
// is treated as special cases.
func ToJsValue(input interface{}) interface{} {

	if input == nil {
		return input
	}
	rv := reflect.ValueOf(input)
	kind := rv.Kind()
	if isPrim(rv) {
		return input
	} else if isPrimPtr(rv) {
		return prvtv(rv.Elem())
	} else if structs.IsStruct(input) {
		// We don't allow methods on these things. TODO think about this
		if rv.NumMethod() != 0 {
			panic("Cannot export structs with methods defined on them through 'ToJsValue'")
		}
		if rv.Kind() == reflect.Ptr {
			if rv.Elem().NumMethod() != 0 {
				panic("Cannot export pointers to structs with methods defined on them through 'ToJsValue' ")
			}
		}
		// This handles both structs and pointers to structs. Once they have 
		// been turned into maps, we need to check the map and validate each value
		// just as normal.
		m := structs.Map(input)
		return ToJsValue(m)
	} else if kind == reflect.Map {
		keys := rv.MapKeys()
		if keys == nil || len(keys) == 0 {
			return make(map[string]interface{})
		}
		if keys[0].Kind() != reflect.String {
			panic("Keys in maps that are exported to the javascript runtime are only allowed to be strings.")
		}
		mp := make(map[string]interface{})
		for _, key := range keys {
			// Call this recursively.
			mp[key.String()] = ToJsValue(ToJsValue(rv.MapIndex(key).Interface()))
		}
		return mp
	} else if kind == reflect.Slice || kind == reflect.Array {
		sl := make([]interface{}, rv.Len())

		for i := 0; i < rv.Len(); i++ {
			// Call this recursively.
			sl[i] = ToJsValue(rv.Index(i).Interface())
		}
		return sl
	} else if kind == reflect.Uintptr || kind == reflect.UnsafePointer {
		panic("uintptrs and unsafe pointers can not be exposed to the javascript runtime.")
	} else if kind == reflect.Complex64 {
		cplx, _ := input.(complex64)
		// Just make maps out of these.
		ret64 := make(map[string]interface{})
		ret64["Real"] = real(cplx)
		ret64["Imag"] = imag(cplx)
		return ret64
	} else if kind == reflect.Complex128 {
		cplx, _ := input.(complex128)
		// Just make maps out of these.
		ret128 := make(map[string]interface{})
		ret128["Real"] = real(cplx)
		ret128["Imag"] = imag(cplx)
		return ret128
	} else if kind == reflect.Ptr {
		// Call this recursively.
		return ToJsValue(rv.Elem().Interface())
	} else {
		panic("Unsupported type: " + rv.Kind().String())
	}
	return nil
}

// Is the value a primitive
func isPrim(v reflect.Value) bool {
	kind := v.Kind()
	return (kind > reflect.Invalid && kind < reflect.Complex64 && !(kind == reflect.Uintptr)) || kind == reflect.String
}

// Is the value a pointer to a primitive
func isPrimPtr(v reflect.Value) bool {
	if v.Kind() == reflect.Ptr {
		return isPrim(v.Elem())
	}
	return false
}

// Primitive reflect value to normal value
func prvtv(val reflect.Value) interface{} {
	k := val.Kind()
	if k == reflect.Bool {
		return val.Bool()
	} else if k == reflect.String {
		return val.String()
	} else if k >= reflect.Int && k <= reflect.Int64 {
		return val.Int()
	} else if k >= reflect.Uint && k <= reflect.Uint64 {
		return val.Uint()
	} else {
		return nil
	}
	return nil
}

// Is the object compatible with otto. In order to be so, it must 
// be a primitive type, a slice/array or a map. The elements of the
// slices and maps must also be of those types, or of course nested.
func IsJsCompat(input interface{}) bool {
	if input == nil {
		return true
	}
	val := reflect.ValueOf(input)
	k := val.Kind()
	if isPrim(val) {
		return true
	} else if k == reflect.Map {
		
		mapKeys := val.MapKeys()
		// We allow empty maps, as they becomes empty objects regardless of key type.
		if len(mapKeys) == 0 {
			return true
		}
		if mapKeys[0].Kind() != reflect.String {
			return false
		}
		for _ , key := range mapKeys {
			if(!IsJsCompat(val.MapIndex(key).Interface()) ) {
				return false
			}
		}
	} else if k == reflect.Array || k == reflect.Slice {
		for i := 0; i < val.Len(); i++ {
			if(!IsJsCompat(val.Index(i).Interface()) ) {
				return false
			}
		}
	} else {
		return false
	}
	return true
}