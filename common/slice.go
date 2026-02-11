package common

import (
	"reflect"
	"sort"
)

// Mapped to AreSimilarSlices
// compare 2 signers lists
// return true if they are same elements, otherwise return false
func AreSimilarSlices(list1 []Address, list2 []Address) bool {
	if len(list1) == 0 && len(list2) == 0 {
		return true
	}
	sort.Slice(list1, func(i, j int) bool {
		return list1[i].String() <= list1[j].String()
	})
	sort.Slice(list2, func(i, j int) bool {
		return list2[i].String() <= list2[j].String()
	})
	return reflect.DeepEqual(list1, list2)
}

// SetSubstract removes all occurrences of items in 'items' from 'array' and returns the new array.
func SetSubstract(array []Address, items []Address) []Address {
	if len(items) == 0 {
		return array
	}

	for _, item := range items {
		for i := len(array) - 1; i >= 0; i-- {
			if array[i] == item {
				array = append(array[:i], array[i+1:]...)
			}
		}
	}

	return array
}

// Return index of the element e in slice s. Return -1 if not found.
func IndexOf(list []Address, x Address) int {
	for i, item := range list {
		if item == x {
			return i
		}
	}
	return -1
}
