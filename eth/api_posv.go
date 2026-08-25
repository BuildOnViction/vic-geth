// Copyright 2015 The go-ethereum Authors
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

package eth

import (
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/sortlgc"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/viction"
	"github.com/ethereum/go-ethereum/rpc"
)

type PublicPosvAPI struct {
	e *Ethereum
}

// NewPublicPosvAPI creates a new PoSV consensus API for full nodes.
func NewPublicPosvAPI(e *Ethereum) *PublicPosvAPI {
	return &PublicPosvAPI{e}
}

// Return validators for given block number.
func (api *PublicPosvAPI) GetValidators(blockNr rpc.BlockNumber) ([]common.Address, error) {
	chainConfig := api.e.BlockChain().Config()
	if chainConfig.Posv == nil || chainConfig.Viction == nil {
		return nil, core.ErrPosvRequired
	}

	var block *types.Block
	if blockNr == rpc.LatestBlockNumber {
		block = api.e.blockchain.CurrentBlock()
	} else {
		block = api.e.blockchain.GetBlockByNumber(uint64(blockNr))
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", blockNr)
	}

	statedb, err := api.e.BlockChain().StateAt(block.Root())
	if err != nil {
		return nil, err
	}
	return viction.GetValidators(chainConfig, chainConfig.Viction, block.Header(), statedb)
}

type PublicPosvDebugAPI struct {
	e *Ethereum
}

// NewPublicPosvDebugAPI creates a new PoSV consensus debug API for archive nodes.
func NewPublicPosvDebugAPI(e *Ethereum) *PublicPosvDebugAPI {
	return &PublicPosvDebugAPI{e}
}

type GetValidatorsResult struct {
	BlockNumber      uint64                          `json:"number"`
	BlockHash        string                          `json:"hash"`
	Candidates       []*GetValidatorsCandidateDetail `json:"candidatess"`
	SortedCandidates []*GetValidatorsCandidateDetail `json:"sortedCandidates"`
	SortType         string                          `json:"sortType"`
	Validators       []*GetValidatorsValidatorDetail `json:"validators"`
	ValidatorLimit   int                             `json:"validatorLimit"`
}

type GetValidatorsCandidateDetail struct {
	Index    int    `json:"index"`
	Address  string `json:"address"`
	Owner    string `json:"owner,omitempty"`
	Capacity string `json:"capacity"`
}

type GetValidatorsValidatorDetail struct {
	Index    int    `json:"index"`
	Address  string `json:"address"`
	Capacity string `json:"capacity"`
}

// Return validators for given block number with detailed steps.
func (api *PublicPosvDebugAPI) GetValidators(blockNr rpc.BlockNumber) (*GetValidatorsResult, error) {
	chainConfig := api.e.BlockChain().Config()
	if chainConfig.Posv == nil || chainConfig.Viction == nil {
		return nil, core.ErrPosvRequired
	}

	result := &GetValidatorsResult{}
	var block *types.Block
	if blockNr == rpc.LatestBlockNumber {
		block = api.e.blockchain.CurrentBlock()
	} else {
		block = api.e.blockchain.GetBlockByNumber(uint64(blockNr))
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", blockNr)
	}
	result.BlockNumber = block.NumberU64()
	result.BlockHash = block.Hash().Hex()

	statedb, err := api.e.BlockChain().StateAt(block.Root())
	if err != nil {
		return nil, err
	}

	candidates := statedb.VicGetCandidates(chainConfig.Viction.ValidatorContract)
	var validators []posv.ValidatorInfo
	result.Candidates = make([]*GetValidatorsCandidateDetail, len(candidates))
	for i, candidate := range candidates {
		owner, cap := statedb.VicGetValidatorInfo(chainConfig.Viction.ValidatorContract, candidate)
		result.Candidates[i] = &GetValidatorsCandidateDetail{
			Index:    i,
			Address:  candidate.Hex(),
			Owner:    owner.Hex(),
			Capacity: cap.String(),
		}
		if !candidate.IsZero() {
			validators = append(validators, posv.ValidatorInfo{Address: candidate, Capacity: cap})
		}
	}

	if chainConfig.IsAtlas(block.Number()) {
		result.SortType = "stable"
		sort.SliceStable(validators, func(i, j int) bool {
			return validators[i].Capacity.Cmp(validators[j].Capacity) >= 0
		})
	} else {
		result.SortType = "legacy"
		sortlgc.Slice(validators, func(i, j int) bool {
			return validators[i].Capacity.Cmp(validators[j].Capacity) >= 0
		})
	}

	result.SortedCandidates = make([]*GetValidatorsCandidateDetail, len(validators))
	for i, v := range validators {
		result.SortedCandidates[i] = &GetValidatorsCandidateDetail{
			Index:    i,
			Address:  v.Address.Hex(),
			Capacity: v.Capacity.String(),
		}
	}

	maxCount := int(chainConfig.Viction.ValidatorMaxCount)
	result.ValidatorLimit = maxCount
	final := validators
	if len(validators) > maxCount {
		final = validators[:maxCount]
	}
	result.Validators = make([]*GetValidatorsValidatorDetail, len(final))
	for i, v := range final {
		result.Validators[i] = &GetValidatorsValidatorDetail{
			Index:    i,
			Address:  v.Address.Hex(),
			Capacity: v.Capacity.String(),
		}
	}

	return result, nil
}
