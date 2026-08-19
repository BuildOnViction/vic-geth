// Copyright 2014 The go-ethereum Authors
// (original work)
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
)

// Apply PoSV-specific hard forks.
func ApplyPosvHardForks(statedb *state.StateDB, config *params.ChainConfig, victionConfig *params.VictionConfig, headerNumber *big.Int) {
	if config.Posv == nil && victionConfig != nil {
		return
	}
	if config.TIPSigningBlock != nil && config.TIPSigningBlock.Cmp(headerNumber) == 0 {
		statedb.RemoveState(victionConfig.ValidatorBlockSignContract)
	}
	if config.AtlasBlock != nil && config.AtlasBlock.Cmp(headerNumber) >= 0 {
		ApplyAtlasHardFork(statedb, victionConfig, config.AtlasBlock, headerNumber)
	}
	if config.SaigonBlock != nil && config.SaigonBlock.Cmp(headerNumber) <= 0 {
		ApplySaigonHardFork(statedb, victionConfig, config.SaigonBlock, headerNumber)
	}
}

// Apply recurring changes of Saigon hard fork:
// - Mint native token to Eco Systtem Fund address.
func ApplySaigonHardFork(statedb *state.StateDB, victionConfig *params.VictionConfig, saigonBlock *big.Int, headBlock *big.Int) {
	if victionConfig.SaigonFundInterval == 0 {
		return
	}
	endBlock := new(big.Int).Add(saigonBlock, new(big.Int).SetUint64(victionConfig.SaigonFundInterval*(victionConfig.SaigonFundRepeat-1)))
	if headBlock.Cmp(saigonBlock) < 0 || headBlock.Cmp(endBlock) > 0 {
		return
	}
	blockOfInterval := new(big.Int).Mod(new(big.Int).Sub(headBlock, saigonBlock), new(big.Int).SetUint64(victionConfig.SaigonFundInterval))
	if blockOfInterval.Cmp(big.NewInt(0)) == 0 {
		if victionConfig.SaigonFundAmount != nil {
			ecoSystemFund := (*big.Int)(victionConfig.SaigonFundAmount)
			statedb.AddBalance(victionConfig.SaigonFundAddress, ecoSystemFund)
		}
	}
}

// Apply one time changes of Atlas hard fork:
// - Set minimum capacity for enrolling zero-gas.
func ApplyAtlasHardFork(statedb *state.StateDB, victionConfig *params.VictionConfig, atlasBlock *big.Int, headBlock *big.Int) {
	if headBlock.Cmp(atlasBlock) == 0 {
		if victionConfig.AtlasVRC25MinCap != nil {
			statedb.VicSetZeroGasMinCap(victionConfig.VRC25Contract, (*big.Int)(victionConfig.AtlasVRC25MinCap))
		}
	}
}

// Check both *from* and *to* addresses are allowed to perform transaction.
func ValidateVictionBlackList(config *params.ChainConfig, from common.Address, to *common.Address) (sender, receiver bool) {
	if config.Viction == nil {
		return false, false
	}
	return config.Viction.IsBlacklisted(from), to != nil && config.Viction.IsBlacklisted(*to)
}
