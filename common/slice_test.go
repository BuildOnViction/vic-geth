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

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAreSimilarSlices(t *testing.T) {
	t.Run("Same elements, different order", func(t *testing.T) {
		slice1 := []int{1, 2, 3, 4, 5}
		slice2 := []int{5, 4, 3, 2, 1}
		assert.True(t, AreSimilarSlices(slice1, slice2))
	})

	t.Run("Different elements", func(t *testing.T) {
		slice1 := []int{1, 2, 3}
		slice2 := []int{1, 2, 4}
		assert.False(t, AreSimilarSlices(slice1, slice2))
	})

	t.Run("Duplicate elements", func(t *testing.T) {
		slice1 := []int{1, 2, 2, 3}
		slice2 := []int{3, 2, 1, 2}
		assert.True(t, AreSimilarSlices(slice1, slice2))
	})

	t.Run("String elements", func(t *testing.T) {
		slice1 := []string{"apple", "banana", "cherry"}
		slice2 := []string{"cherry", "apple", "banana"}
		assert.True(t, AreSimilarSlices(slice1, slice2))
	})
}

func TestIndexOf(t *testing.T) {
	t.Run("IntSlice_ElementFound", func(t *testing.T) {
		slice := []int{10, 20, 30, 40, 50}
		assert.Equal(t, 0, IndexOf(slice, 10), "First element")
		assert.Equal(t, 2, IndexOf(slice, 30), "Middle element")
		assert.Equal(t, 4, IndexOf(slice, 50), "Last element")
	})

	t.Run("IntSlice_ElementNotFound", func(t *testing.T) {
		slice := []int{10, 20, 30, 40, 50}
		assert.Equal(t, -1, IndexOf(slice, 100), "Element not in slice")
		assert.Equal(t, -1, IndexOf(slice, 0), "Element smaller than all")
	})

	t.Run("IntSlice_EmptySlice", func(t *testing.T) {
		var slice []int
		assert.Equal(t, -1, IndexOf(slice, 10), "Empty slice should return -1")
	})

	t.Run("IntSlice_DuplicateElements", func(t *testing.T) {
		slice := []int{5, 10, 5, 20, 5}
		assert.Equal(t, 0, IndexOf(slice, 5), "Should return first occurrence")
	})

	t.Run("StringSlice_ElementFound", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry", "date"}
		assert.Equal(t, 0, IndexOf(slice, "apple"), "First element")
		assert.Equal(t, 2, IndexOf(slice, "cherry"), "Middle element")
		assert.Equal(t, 3, IndexOf(slice, "date"), "Last element")
	})

	t.Run("StringSlice_ElementNotFound", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry"}
		assert.Equal(t, -1, IndexOf(slice, "grape"), "Element not in slice")
		assert.Equal(t, -1, IndexOf(slice, ""), "Empty string not in slice")
	})

	t.Run("StringSlice_EmptySlice", func(t *testing.T) {
		var slice []string
		assert.Equal(t, -1, IndexOf(slice, "test"), "Empty slice should return -1")
	})

	t.Run("StringSlice_EmptyStringElement", func(t *testing.T) {
		slice := []string{"apple", "", "banana"}
		assert.Equal(t, 1, IndexOf(slice, ""), "Empty string in slice")
	})

	t.Run("BoolSlice", func(t *testing.T) {
		slice := []bool{true, false, true, true}
		assert.Equal(t, 0, IndexOf(slice, true), "First true")
		assert.Equal(t, 1, IndexOf(slice, false), "First false")
	})

	t.Run("SingleElement", func(t *testing.T) {
		slice := []int{42}
		assert.Equal(t, 0, IndexOf(slice, 42), "Single element found")
		assert.Equal(t, -1, IndexOf(slice, 100), "Single element not found")
	})

	t.Run("Float64Slice", func(t *testing.T) {
		slice := []float64{1.5, 2.7, 3.9, 4.2}
		assert.Equal(t, 1, IndexOf(slice, 2.7), "Float element found")
		assert.Equal(t, -1, IndexOf(slice, 5.5), "Float element not found")
	})
}

func TestSetSubstract(t *testing.T) {
	t.Run("IntSlice_RemoveSomeElements", func(t *testing.T) {
		a := []int{1, 5, 4, 3, 5, 2}
		b := []int{3, 4, 5}
		result := SetSubstract(a, b)
		expected := []int{1, 2}
		assert.True(t, AreSimilarSlices(result, expected))
	})

	t.Run("IntSlice_RemoveNoElements", func(t *testing.T) {
		a := []int{1, 2, 3}
		b := []int{4, 5, 6}
		result := SetSubstract(a, b)
		expected := []int{1, 2, 3}
		assert.True(t, AreSimilarSlices(result, expected))
	})

	t.Run("IntSlice_RemoveAllElements", func(t *testing.T) {
		a := []int{1, 2, 3}
		b := []int{1, 2, 3, 4, 5}
		result := SetSubstract(a, b)
		assert.Empty(t, result)
	})

	t.Run("IntSlice_EmptyFirstSlice", func(t *testing.T) {
		var a []int
		b := []int{1, 2, 3}
		result := SetSubstract(a, b)
		assert.Empty(t, result)
	})

	t.Run("IntSlice_EmptySecondSlice", func(t *testing.T) {
		a := []int{1, 2, 3}
		var b []int
		result := SetSubstract(a, b)
		expected := []int{1, 2, 3}
		assert.True(t, AreSimilarSlices(result, expected))
	})

	t.Run("IntSlice_BothEmpty", func(t *testing.T) {
		var a, b []int
		result := SetSubstract(a, b)
		assert.Empty(t, result)
	})

	t.Run("IntSlice_WithDuplicates", func(t *testing.T) {
		a := []int{1, 2, 2, 3, 3, 4}
		b := []int{2, 3}
		result := SetSubstract(a, b)
		// Should return unique elements from a not in b
		expected := []int{1, 4}
		assert.True(t, AreSimilarSlices(result, expected))
	})

	t.Run("StringSlice_RemoveSomeElements", func(t *testing.T) {
		a := []string{"apple", "banana", "cherry", "date"}
		b := []string{"banana", "date"}
		result := SetSubstract(a, b)
		expected := []string{"apple", "cherry"}
		assert.True(t, AreSimilarSlices(result, expected))
	})

	t.Run("StringSlice_RemoveNoElements", func(t *testing.T) {
		a := []string{"apple", "banana"}
		b := []string{"cherry", "date"}
		result := SetSubstract(a, b)
		expected := []string{"apple", "banana"}
		assert.True(t, AreSimilarSlices(result, expected))
	})

	t.Run("SingleElementRemaining", func(t *testing.T) {
		a := []int{1, 2, 3}
		b := []int{1, 2}
		result := SetSubstract(a, b)
		expected := []int{3}
		assert.Equal(t, expected, result)
	})
}
