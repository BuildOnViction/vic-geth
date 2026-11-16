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

package posv

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestFinalityTallyHasQuorum(t *testing.T) {
	validators := makeFinalityTestAddresses(8)
	outsider := common.HexToAddress("0xdead")

	tests := []struct {
		name       string
		validators []common.Address
		signers    []common.Address
		want       bool
	}{
		{
			name:       "five of eight validators is below 75 percent",
			validators: validators,
			signers:    validators[:5],
			want:       false,
		},
		{
			name:       "six of eight validators reaches 75 percent",
			validators: validators,
			signers:    validators[:6],
			want:       true,
		},
		{
			name:       "all validators reach quorum",
			validators: validators,
			signers:    validators,
			want:       true,
		},
		{
			name:       "repeated signer counts once",
			validators: validators,
			signers:    append(append([]common.Address{}, validators[:5]...), validators[4]),
			want:       false,
		},
		{
			name:       "signer outside the validator set is not counted",
			validators: validators,
			signers:    append(append([]common.Address{}, validators[:5]...), outsider),
			want:       false,
		},
		{
			name:       "empty validator set never reaches quorum",
			validators: nil,
			signers:    []common.Address{outsider},
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tally := NewFinalityTally(15, tt.validators)

			for _, signer := range tt.signers {
				tally.AddSigner(signer)
			}

			if have := tally.HasQuorum(); have != tt.want {
				t.Fatalf("have %v, want %v", have, tt.want)
			}
		})
	}
}

func makeFinalityTestAddresses(count int) []common.Address {
	addrs := make([]common.Address, count)
	for i := range addrs {
		addrs[i] = common.BytesToAddress([]byte{byte(i + 1)})
	}
	return addrs
}
