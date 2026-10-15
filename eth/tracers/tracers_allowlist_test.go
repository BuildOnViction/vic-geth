// Copyright 2025 The Viction Authors
// (modifications)
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

package tracers

import "testing"

// bundledTracers lists the tracers embedded from internal/tracers. Keeping an
// explicit copy catches accidental additions or removals of trusted code.
var bundledTracers = []string{
	"4byteTracer",
	"bigramTracer",
	"callTracer",
	"evmdisTracer",
	"noopTracer",
	"opcountTracer",
	"prestateTracer",
	"trigramTracer",
	"unigramTracer",
}

func TestIsBuiltinAcceptsBundledTracers(t *testing.T) {
	for _, name := range bundledTracers {
		if !IsBuiltin(name) {
			t.Errorf("bundled tracer %q rejected", name)
		}
	}
}

func TestIsBuiltinMatchesBundledSet(t *testing.T) {
	if len(all) != len(bundledTracers) {
		t.Errorf("bundled tracer count mismatch: have %d, want %d", len(all), len(bundledTracers))
	}
}

func TestIsBuiltinRejectsNonBundledNames(t *testing.T) {
	rejected := []string{
		"",
		"unknownTracer",
		"CallTracer",
		"calltracer",
		" callTracer",
		"callTracer ",
		"callTracer\n",
		"callTracer\x00",
		"callTracerX",
		"callTracer;1",
		"{step:function(){},fault:function(){},result:function(){return 1}}",
		`{step:function(){},fault:function(){},result:function(){return this["constructor"]}}`,
	}
	for _, name := range rejected {
		if IsBuiltin(name) {
			t.Errorf("non-bundled tracer %q accepted", name)
		}
	}
}
