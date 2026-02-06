// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"reflect"
	"sort"
)

// Mapped to AreSimilarSlices
// compare 2 signers lists
// return true if they are same elements, otherwise return false
func CompareSignersLists(list1 []Address, list2 []Address) bool {
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

// Extract validators from byte array.
func RemoveItemFromArray(array []Address, items []Address) []Address {
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
