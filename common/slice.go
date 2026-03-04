// Copyright 2025 The Viction Authors
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

// Compare two slices have same elements but maynot in same order.
func AreSimilarSlices[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	itemCounts := make(map[T]int)
	for _, item := range a {
		itemCounts[item]++
	}
	for _, item := range b {
		if itemCounts[item] == 0 {
			return false
		}
		itemCounts[item]--
	}
	return true
}

// Return index of the element e in slice s. Return -1 if not found.
func IndexOf[T comparable](s []T, e T) int {
	for i, item := range s {
		if item == e {
			return i
		}
	}
	return -1
}

// Return a new slice with elements that belong to set a but not set b.
func SetSubstract[T comparable](a, b []T) []T {
	bmap := map[T]bool{}
	for _, item := range b {
		bmap[item] = true
	}
	rmap := map[T]bool{}
	res := []T{}
	for _, item := range a {
		if _, ok := bmap[item]; ok {
			continue
		}
		if _, ok := rmap[item]; !ok {
			rmap[item] = true
			res = append(res, item)
		}
	}
	return res
}
