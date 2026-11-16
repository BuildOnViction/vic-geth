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

package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestReceiptIsSuccessful(t *testing.T) {
	tests := []struct {
		name    string
		receipt *Receipt
		want    bool
	}{
		{
			name:    "successful status",
			receipt: &Receipt{Status: ReceiptStatusSuccessful},
			want:    true,
		},
		{
			name:    "failed status",
			receipt: &Receipt{Status: ReceiptStatusFailed},
			want:    false,
		},
		{
			name:    "pre-Byzantium post-state root counts as successful",
			receipt: &Receipt{PostState: common.HexToHash("0x01").Bytes(), Status: ReceiptStatusFailed},
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if have := tt.receipt.IsSuccessful(); have != tt.want {
				t.Fatalf("have %v, want %v", have, tt.want)
			}
		})
	}
}
