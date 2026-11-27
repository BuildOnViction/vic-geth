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

package misc

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
)

// A chain config with Viction fork blocks but no viction section must skip the
// Viction hard forks instead of dereferencing the nil VictionConfig.
func TestApplyPosvHardForksNilViction(t *testing.T) {
	posv := &params.PosvConfig{Period: 2, Epoch: 900, Gap: 5}
	tests := []struct {
		name   string
		config *params.ChainConfig
	}{
		{"posv, saigonBlock", &params.ChainConfig{Posv: posv, SaigonBlock: big.NewInt(10)}},
		{"posv, tipSigningBlock", &params.ChainConfig{Posv: posv, TIPSigningBlock: big.NewInt(10)}},
		{"posv, atlasBlock", &params.ChainConfig{Posv: posv, AtlasBlock: big.NewInt(10)}},
		{"no posv, saigonBlock", &params.ChainConfig{SaigonBlock: big.NewInt(10)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panicked at the fork block: %v", r)
				}
			}()
			ApplyPosvHardForks(statedb, tt.config, tt.config.Viction, big.NewInt(10))
		})
	}
}

// With a viction section the hard forks still apply: the Saigon fund is minted
// at the Saigon block.
func TestApplyPosvHardForksSaigonFund(t *testing.T) {
	fund := common.HexToAddress("0x00000000000000000000000000000000000000ee")
	amount := big.NewInt(1000)
	config := &params.ChainConfig{
		Posv:        &params.PosvConfig{Period: 2, Epoch: 900, Gap: 5},
		SaigonBlock: big.NewInt(10),
		Viction: &params.VictionConfig{
			SaigonFundInterval: 10,
			SaigonFundRepeat:   1,
			SaigonFundAddress:  fund,
			SaigonFundAmount:   (*math.Decimal256)(amount),
		},
	}
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	ApplyPosvHardForks(statedb, config, config.Viction, big.NewInt(10))
	if got := statedb.GetBalance(fund); got.Cmp(amount) != 0 {
		t.Fatalf("fund balance = %v, want %v", got, amount)
	}
}
