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

package params

import (
	"math/big"
	"strings"
	"testing"
)

func TestCheckConfigForkOrderVictionConfig(t *testing.T) {
	posv := &PosvConfig{Period: 2, Epoch: 900, Gap: 5}
	tests := []struct {
		name    string
		config  *ChainConfig
		wantErr string // empty means the config is valid
	}{
		{"viction", VictionChainConfig, ""},
		{"victest", VictestChainConfig, ""},
		{"ethereum mainnet", MainnetChainConfig, ""},
		{"goerli", GoerliChainConfig, ""},
		{"all ethash", AllEthashProtocolChanges, ""},
		{"all clique", AllCliqueProtocolChanges, ""},
		{"test chain", TestChainConfig, ""},
		{"posv without viction", &ChainConfig{Posv: posv}, "posv is set but viction is missing"},
		{"viction without posv", &ChainConfig{Viction: &VictionConfig{}}, "viction is set but posv is missing"},
		{"viction fork without posv and viction", &ChainConfig{SaigonBlock: big.NewInt(10), AtlasBlock: big.NewInt(20)}, "Viction forks [atlasBlock saigonBlock] are set"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.CheckConfigForkOrder()
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("valid config rejected: %v", err)
			case tt.wantErr != "" && err == nil:
				t.Fatalf("invalid config accepted, want error containing %q", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Fatalf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
