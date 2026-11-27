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

package victionapi

import (
	"testing"
)

// makeSecret encrypts text with opening and stores it right-aligned in 32 bytes,
// as validators submit it to the randomize contract.
func makeSecret(t *testing.T, opening [32]byte, text string) [32]byte {
	enc, err := EncryptAesCfb(opening[:], text)
	if err != nil {
		t.Fatal(err)
	}
	var secret [32]byte
	copy(secret[32-len(enc):], enc)
	return secret
}

// A secret that cannot be decrypted or is not an integer is skipped, like the
// legacy client does, instead of failing the attestor computation.
func TestDecryptRandomizeSkipsInvalidSecrets(t *testing.T) {
	var opening [32]byte
	copy(opening[:], "0123456789abcdef0123456789abcdef")
	valid := makeSecret(t, opening, "42")
	notInt := makeSecret(t, opening, "abc")
	var short [32]byte
	short[31] = 0x01 // too short to hold the AES IV

	tests := []struct {
		name    string
		secrets [][32]byte
		want    int64
	}{
		{"valid", [][32]byte{valid}, 42},
		{"short before valid", [][32]byte{short, valid}, 42},
		{"short after valid", [][32]byte{valid, short}, 42},
		{"only short", [][32]byte{short}, 0},
		{"not an integer", [][32]byte{valid, notInt}, 42},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecryptRandomize(tt.secrets, opening)
			if err != nil {
				t.Fatalf("DecryptRandomize error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("DecryptRandomize = %d, want %d", got, tt.want)
			}
		})
	}
}
