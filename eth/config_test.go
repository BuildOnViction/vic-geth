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

package eth

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

func TestSkipCompatRewindFor(t *testing.T) {
	explicitSkip := new(bool)
	*explicitSkip = true
	explicitRewind := new(bool)
	*explicitRewind = false

	tests := []struct {
		name    string
		config  Config
		genesis common.Hash
		want    bool
	}{
		{"unset viction genesis", Config{}, params.VictionGenesisHash, true},
		{"unset victest genesis", Config{}, params.VictestGenesisHash, true},
		{"unset mainnet genesis", Config{}, params.MainnetGenesisHash, false},
		{"unset unknown genesis", Config{}, common.HexToHash("0xfeedbeef"), false},
		{"explicit false wins on viction", Config{SkipCompatRewind: explicitRewind}, params.VictionGenesisHash, false},
		{"explicit true wins on mainnet", Config{SkipCompatRewind: explicitSkip}, params.MainnetGenesisHash, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.SkipCompatRewindFor(tt.genesis); got != tt.want {
				t.Errorf("SkipCompatRewindFor() = %v, want %v", got, tt.want)
			}
		})
	}
}
