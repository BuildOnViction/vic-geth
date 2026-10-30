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
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/sortlgc"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/victionapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// PublicPosvAPI is the API exclusive to PoSV consensus.
type PublicPosvAPI struct {
	e *Ethereum
}

// NewPublicPosvAPI creates a new PoSV consensus API for full nodes.
func NewPublicPosvAPI(e *Ethereum) *PublicPosvAPI {
	return &PublicPosvAPI{e}
}

// Return the epoch reward breakdown for the checkpoint block identified by hash.
// The block must be a checkpoint block (number divisible by the epoch size).
// Passing a non-checkpoint hash returns an error.
func (api *PublicPosvAPI) GetRewardByHash(hash common.Hash) (*posv.EpochReward, error) {
	return getRewardByCheckpointHeader(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithHash(hash, false)))
}

// Return the epoch reward breakdown for the checkpoint block identified by either a block hash or a block number.
// The block must be a checkpoint block (number divisible by the epoch size).
// Passing a non-checkpoint identifier returns an error.
func (api *PublicPosvAPI) GetRewardByHashOrNumber(blockNrOrHash rpc.BlockNumberOrHash) (*posv.EpochReward, error) {
	return getRewardByCheckpointHeader(api.e, headerByNumberOrHash(api.e, blockNrOrHash))
}

// Return the epoch reward breakdown for the checkpoint block at the given number.
// The block must be a checkpoint block (number divisible by the epoch size).
// Passing a non-checkpoint number returns an error.
func (api *PublicPosvAPI) GetRewardByNumber(blockNr rpc.BlockNumber) (*posv.EpochReward, error) {
	return getRewardByCheckpointHeader(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithNumber(blockNr)))
}

// Return the validator pairs (validator-attestor) effective at the block identified by hash.
// The pairs are derived from the checkpoint header governing the block's epoch.
func (api *PublicPosvAPI) GetValidatorPairsByHash(hash common.Hash) (map[common.Address]common.Address, error) {
	pairs, _, _, err := getValidatorPairsByCheckpointHeader(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithHash(hash, false)))
	return pairs, err
}

// Return the validator pairs (validator-attestor) effective at the block identified by either a block hash or a block number.
// The pairs are derived from the checkpoint header governing the block's epoch.
func (api *PublicPosvAPI) GetValidatorPairsByHashOrNumber(blockNrOrHash rpc.BlockNumberOrHash) (map[common.Address]common.Address, error) {
	pairs, _, _, err := getValidatorPairsByCheckpointHeader(api.e, headerByNumberOrHash(api.e, blockNrOrHash))
	return pairs, err
}

// Return the validator pairs (validator-attestor) effective at the block at the given number.
// The pairs are derived from the checkpoint header governing the block's epoch.
func (api *PublicPosvAPI) GetValidatorPairsByNumber(blockNr rpc.BlockNumber) (map[common.Address]common.Address, error) {
	pairs, _, _, err := getValidatorPairsByCheckpointHeader(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithNumber(blockNr)))
	return pairs, err
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
	return victionapi.GetValidators(block.Header(), chainConfig, chainConfig.Viction, statedb)
}

// PublicPosvDebugAPI mirrors PublicPosvAPI with detailed calculation steps that are useful for debugging.
type PublicPosvDebugAPI struct {
	e *Ethereum
}

// NewPublicPosvDebugAPI creates a new PoSV consensus debug API for archive nodes.
func NewPublicPosvDebugAPI(e *Ethereum) *PublicPosvDebugAPI {
	return &PublicPosvDebugAPI{e}
}

// GetRewardResult stores the epoch reward breakdown with the block and epoch context it was computed at.
type GetRewardResult struct {
	BlockNumber        uint64                      `json:"number"`
	BlockHash          string                      `json:"hash"`
	Epoch              uint64                      `json:"epoch"`
	EpochSize          uint64                      `json:"epochSize"`
	Reward             *posv.EpochReward           `json:"reward"`
	StakeholderRewards map[common.Address]*big.Int `json:"stakeholderRewards"` // exposed here; EpochReward hides them (json:"-")
}

// GetValidatorPairsResult stores the validator pairs (validator-attestor) with the block and epoch context it was computed at,
// together with the checkpoint header and rotation offset it was derived from.
type GetValidatorPairsResult struct {
	BlockNumber      uint64                            `json:"number"`
	BlockHash        string                            `json:"hash"`
	Epoch            uint64                            `json:"epoch"`
	EpochSize        uint64                            `json:"epochSize"`
	CheckpointNumber uint64                            `json:"checkpointNumber"`
	CheckpointHash   string                            `json:"checkpointHash"`
	Offset           uint64                            `json:"offset"`
	Pairs            map[common.Address]common.Address `json:"pairs"`
}

type GetValidatorsResult struct {
	BlockNumber      uint64                          `json:"number"`
	BlockHash        string                          `json:"hash"`
	Candidates       []*GetValidatorsCandidateDetail `json:"candidates"`
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

// Return the epoch reward breakdown for the checkpoint block identified by hash,
// with the block and epoch context it was computed at.
// The block must be a checkpoint block (number divisible by the epoch size).
// Passing a non-checkpoint hash returns an error.
func (api *PublicPosvDebugAPI) GetRewardByHash(hash common.Hash) (*GetRewardResult, error) {
	return getRewardByCheckpointHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithHash(hash, false)))
}

// Return the epoch reward breakdown for the checkpoint block identified by either a block hash or a block number,
// with the block and epoch context it was computed at.
// The block must be a checkpoint block (number divisible by the epoch size).
// Passing a non-checkpoint identifier returns an error.
func (api *PublicPosvDebugAPI) GetRewardByHashOrNumber(blockNrOrHash rpc.BlockNumberOrHash) (*GetRewardResult, error) {
	return getRewardByCheckpointHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, blockNrOrHash))
}

// Return the epoch reward breakdown for the checkpoint block at the given number,
// with the block and epoch context it was computed at.
// The block must be a checkpoint block (number divisible by the epoch size).
// Passing a non-checkpoint number returns an error.
func (api *PublicPosvDebugAPI) GetRewardByNumber(blockNr rpc.BlockNumber) (*GetRewardResult, error) {
	return getRewardByCheckpointHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithNumber(blockNr)))
}

// Return the validator pairs (validator-attestor) effective at the block identified by hash,
// with the block and epoch context and the derivation inputs it was computed from.
// The pairs are derived from the checkpoint header governing the block's epoch.
func (api *PublicPosvDebugAPI) GetValidatorPairsByHash(hash common.Hash) (*GetValidatorPairsResult, error) {
	return getValidatorPairsByCheckpointHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithHash(hash, false)))
}

// Return the validator pairs (validator-attestor) effective at the block identified by either a block hash or a block number,
// with the block and epoch context and the derivation inputs it was computed from.
// The pairs are derived from the checkpoint header governing the block's epoch.
func (api *PublicPosvDebugAPI) GetValidatorPairsByHashOrNumber(blockNrOrHash rpc.BlockNumberOrHash) (*GetValidatorPairsResult, error) {
	return getValidatorPairsByCheckpointHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, blockNrOrHash))
}

// Return the validator pairs (validator-attestor) effective at the block at the given number,
// with the block and epoch context and the derivation inputs it was computed from.
// The pairs are derived from the checkpoint header governing the block's epoch.
func (api *PublicPosvDebugAPI) GetValidatorPairsByNumber(blockNr rpc.BlockNumber) (*GetValidatorPairsResult, error) {
	return getValidatorPairsByCheckpointHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithNumber(blockNr)))
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

// Resolve a block number or hash to a header:
//   - "latest" resolves to the current chain head.
//   - Any other number to the canonical chain.
//   - When no header matches return nil.
func headerByNumberOrHash(e *Ethereum, blockNrOrHash rpc.BlockNumberOrHash) *types.Header {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		if blockNr == rpc.LatestBlockNumber {
			return e.blockchain.CurrentHeader()
		}
		return e.blockchain.GetHeaderByNumber(uint64(blockNr))
	}
	if blockHash, ok := blockNrOrHash.Hash(); ok {
		return e.blockchain.GetHeaderByHash(blockHash)
	}
	return nil
}

// Compute the epoch reward for a checkpoint header.
// Return an error when the header is missing, is not a checkpoint block, or its state is unavailable.
func getRewardByCheckpointHeader(e *Ethereum, header *types.Header) (*posv.EpochReward, error) {
	if header == nil {
		return nil, errors.New("header at block number or hash is not found")
	}
	cfg := e.blockchain.Config()
	if cfg.Posv == nil || cfg.Viction == nil {
		return nil, core.ErrPosvRequired
	}
	if !cfg.Posv.IsCheckpointBlock(header.Number.Uint64()) {
		return nil, errors.New("header is not a checkpoint block")
	}
	if _, ok := e.engine.(*posv.Posv); !ok {
		return nil, errors.New("engine is not a posv engine")
	}
	statedb, err := e.blockchain.StateAt(header.Root)
	if err != nil {
		log.Info("[PosvAPI] Failed to get state at checkpoint", "hash", header.Hash(), "err", err)
		return nil, err
	}
	epochReward, err := e.PosvGetEpochRewards(header, cfg, cfg.Posv, cfg.Viction, e.blockchain, statedb, log.Root())
	if err != nil {
		return nil, err
	}
	if epochReward == nil {
		return nil, errors.New("epoch reward is nil")
	}
	return epochReward, nil
}

// Compute the epoch reward for a checkpoint header and wrap it with the block and epoch context for the posvdebug API.
func getRewardByCheckpointHeaderWithDebugInfo(e *Ethereum, header *types.Header) (*GetRewardResult, error) {
	epochReward, err := getRewardByCheckpointHeader(e, header)
	if err != nil {
		return nil, err
	}
	cfg := e.blockchain.Config()
	return &GetRewardResult{
		BlockNumber:        header.Number.Uint64(),
		BlockHash:          header.Hash().Hex(),
		Epoch:              header.Number.Uint64() / cfg.Posv.Epoch,
		EpochSize:          cfg.Posv.Epoch,
		Reward:             epochReward,
		StakeholderRewards: epochReward.StakeholderRewards,
	}, nil
}

// Compute the validator pairs (validator-attestor) effective at the given header,
// together with the rotation offset and the checkpoint header they were derived from.
// Return an error when the header is missing, its checkpoint header is missing, or the pairs cannot be derived.
func getValidatorPairsByCheckpointHeader(e *Ethereum, header *types.Header) (map[common.Address]common.Address, uint64, *types.Header, error) {
	if header == nil {
		return nil, 0, nil, errors.New("header at block number or hash is not found")
	}
	cfg := e.blockchain.Config()
	if cfg.Posv == nil || cfg.Viction == nil {
		return nil, 0, nil, core.ErrPosvRequired
	}
	if _, ok := e.engine.(*posv.Posv); !ok {
		return nil, 0, nil, errors.New("engine is not a posv engine")
	}
	checkpointHeader := posv.GetCheckpointHeader(cfg.Posv, header, nil, e.blockchain)
	if checkpointHeader == nil {
		return nil, 0, nil, errors.New("checkpoint header not found")
	}
	pairs, offset, err := e.PosvGetCreatorAttestorPairs(header, checkpointHeader, cfg, cfg.Posv, cfg.Viction)
	if err != nil {
		return nil, 0, nil, err
	}
	return pairs, offset, checkpointHeader, nil
}

// Compute the validator pairs (validator-attestor) for a header and wrap them with the block and epoch context for the debugging.
func getValidatorPairsByCheckpointHeaderWithDebugInfo(e *Ethereum, header *types.Header) (*GetValidatorPairsResult, error) {
	pairs, offset, checkpointHeader, err := getValidatorPairsByCheckpointHeader(e, header)
	if err != nil {
		return nil, err
	}
	cfg := e.blockchain.Config()
	return &GetValidatorPairsResult{
		BlockNumber:      header.Number.Uint64(),
		BlockHash:        header.Hash().Hex(),
		Epoch:            header.Number.Uint64() / cfg.Posv.Epoch,
		EpochSize:        cfg.Posv.Epoch,
		CheckpointNumber: checkpointHeader.Number.Uint64(),
		CheckpointHash:   checkpointHeader.Hash().Hex(),
		Offset:           offset,
		Pairs:            pairs,
	}, nil
}
