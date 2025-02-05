package elementsmatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// isEmpty gets whether the specified object is considered empty or not.
func isEmpty(object interface{}) bool {

	// get nil case out of the way
	if object == nil {
		return true
	}

	objValue := reflect.ValueOf(object)

	switch objValue.Kind() {
	// collection types are empty when they have no element
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		return objValue.Len() == 0
		// pointers are empty if nil or if the value they point to is empty
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		deref := objValue.Elem().Interface()
		return isEmpty(deref)
		// for all other types, compare against the zero value
	default:
		zero := reflect.Zero(objValue.Type())
		return reflect.DeepEqual(object, zero.Interface())
	}
}

// ObjectsAreEqual determines if two objects are considered equal.
//
// This function does no assertion of any kind.
func ObjectsAreEqual(expected, actual interface{}) bool {
	if expected == nil || actual == nil {
		return expected == actual
	}

	exp, ok := expected.([]byte)
	if !ok {
		return reflect.DeepEqual(expected, actual)
	}

	act, ok := actual.([]byte)
	if !ok {
		return false
	}
	if exp == nil || act == nil {
		return exp == nil && act == nil
	}
	return bytes.Equal(exp, act)
}

// ElementsMatch determines if two lists contain the same elements, regardless of order.
func ElementsMatch(listA, listB interface{}) (ok bool, err error) {
	if isEmpty(listA) && isEmpty(listB) {
		return true, nil
	}

	aType := reflect.TypeOf(listA)
	if aType == nil {
		return false, fmt.Errorf("listA is nil")
	}
	bType := reflect.TypeOf(listB)
	if bType == nil {
		return false, fmt.Errorf("listB is nil")
	}

	if aType.Kind() != reflect.Array && aType.Kind() != reflect.Slice {
		return false, fmt.Errorf("%q has an unsupported type %s", listA, aType.Kind())
	}

	if bType.Kind() != reflect.Array && bType.Kind() != reflect.Slice {
		return false, fmt.Errorf("%q has an unsupported type %s", listB, bType.Kind())
	}

	aValue := reflect.ValueOf(listA)
	bValue := reflect.ValueOf(listB)

	aLen := aValue.Len()
	bLen := bValue.Len()

	if aLen != bLen {
		return false, nil
	}

	// Mark indexes in bValue that we already used
	visited := make([]bool, bLen)
	for i := 0; i < aLen; i++ {
		element := aValue.Index(i).Interface()
		found := false
		for j := 0; j < bLen; j++ {
			if visited[j] {
				continue
			}
			var e1, e2 interface{}
			b1, err := json.Marshal(element)
			if err != nil {
				return false, err
			}
			err = json.Unmarshal(b1, &e1)
			if err != nil {
				return false, err
			}
			b2, err := json.Marshal(bValue.Index(j).Interface())
			if err != nil {
				return false, err
			}
			err = json.Unmarshal(b2, &e2)
			if err != nil {
				return false, err
			}

			if ObjectsAreEqual(e1, e2) {
				visited[j] = true
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}

	return true, nil
}
