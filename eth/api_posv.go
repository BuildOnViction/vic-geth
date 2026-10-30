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

// Return the finality percentage of the block identified by hash.
// The finality is the share of the governing validator set that signed the block, raised to the finality of the nearest finalized block ahead on the same path.
func (api *PublicPosvAPI) GetBlockFinalityByHash(hash common.Hash) (uint, error) {
	finality, _, _, _, err := getBlockFinalityByHeader(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithHash(hash, false)))
	return finality, err
}

// Return the finality percentage of the block identified by either a block hash or a block number.
// The finality is the share of the governing validator set that signed the block, raised to the finality of the nearest finalized block ahead on the same path.
func (api *PublicPosvAPI) GetBlockFinalityByHashOrNumber(blockNrOrHash rpc.BlockNumberOrHash) (uint, error) {
	finality, _, _, _, err := getBlockFinalityByHeader(api.e, headerByNumberOrHash(api.e, blockNrOrHash))
	return finality, err
}

// Return the finality percentage of the block at the given number.
// The finality is the share of the governing validator set that signed the block, raised to the finality of the nearest finalized block ahead on the same path.
func (api *PublicPosvAPI) GetBlockFinalityByNumber(blockNr rpc.BlockNumber) (uint, error) {
	finality, _, _, _, err := getBlockFinalityByHeader(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithNumber(blockNr)))
	return finality, err
}

// Return the validators that signed the block identified by hash.
// The signers are the creator and attestor of the given block plus unique senders of successful BlockSigner transactions for that block; all are restricted to the epoch's validator set.
func (api *PublicPosvAPI) GetBlockSignersByHash(hash common.Hash) ([]common.Address, error) {
	signers, _, _, err := getBlockSignersByHeader(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithHash(hash, false)))
	return signers, err
}

// Return the validators that signed the block identified by either a block hash or a block number.
// The signers are the creator and attestor of the given block plus unique senders of successful BlockSigner transactions for that block; all are restricted to the epoch's validator set.
func (api *PublicPosvAPI) GetBlockSignersByHashOrNumber(blockNrOrHash rpc.BlockNumberOrHash) ([]common.Address, error) {
	signers, _, _, err := getBlockSignersByHeader(api.e, headerByNumberOrHash(api.e, blockNrOrHash))
	return signers, err
}

// Return the validators that signed the block at the given number.
// The signers are the creator and attestor of the given block plus unique senders of successful BlockSigner transactions for that block; all are restricted to the epoch's validator set.
func (api *PublicPosvAPI) GetBlockSignersByNumber(blockNr rpc.BlockNumber) ([]common.Address, error) {
	signers, _, _, err := getBlockSignersByHeader(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithNumber(blockNr)))
	return signers, err
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

// GetBlockFinalityResult stores the block finality with the block and epoch context it was computed at,
// together with the signer share and the nearest finalized block ahead it was reconciled with.
type GetBlockFinalityResult struct {
	BlockNumber              uint64 `json:"number"`
	BlockHash                string `json:"hash"`
	Epoch                    uint64 `json:"epoch"`
	EpochSize                uint64 `json:"epochSize"`
	Finality                 uint   `json:"finality"`
	SignerFinality           uint   `json:"signerFinality"`
	ClosestFinalizedNumber   uint64 `json:"closestFinalizedNumber,omitempty"`
	ClosestFinalizedHash     string `json:"closestFinalizedHash,omitempty"`
	ClosestFinalizedFinality uint   `json:"closestFinalizedFinality,omitempty"`
}

// GetBlockSignersResult stores the block signers with the block and epoch context it was computed at,
// together with the signed block and validator set they were derived from.
type GetBlockSignersResult struct {
	BlockNumber       uint64           `json:"number"`
	BlockHash         string           `json:"hash"`
	Epoch             uint64           `json:"epoch"`
	EpochSize         uint64           `json:"epochSize"`
	SignedBlockNumber uint64           `json:"signedBlockNumber"`
	SignedBlockHash   string           `json:"signedBlockHash"`
	Validators        []common.Address `json:"validators"`
	Signers           []common.Address `json:"signers"`
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

// Return the finality percentage of the block identified by hash,
// with the block and epoch context and the reconciliation steps it was computed from.
// The finality is the share of the governing validator set that signed the block, raised to the finality of the nearest finalized block ahead on the same path.
func (api *PublicPosvDebugAPI) GetBlockFinalityByHash(hash common.Hash) (*GetBlockFinalityResult, error) {
	return getBlockFinalityByHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithHash(hash, false)))
}

// Return the finality percentage of the block identified by either a block hash or a block number,
// with the block and epoch context and the reconciliation steps it was computed from.
// The finality is the share of the governing validator set that signed the block, raised to the finality of the nearest finalized block ahead on the same path.
func (api *PublicPosvDebugAPI) GetBlockFinalityByHashOrNumber(blockNrOrHash rpc.BlockNumberOrHash) (*GetBlockFinalityResult, error) {
	return getBlockFinalityByHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, blockNrOrHash))
}

// Return the finality percentage of the block at the given number,
// with the block and epoch context and the reconciliation steps it was computed from.
// The finality is the share of the governing validator set that signed the block, raised to the finality of the nearest finalized block ahead on the same path.
func (api *PublicPosvDebugAPI) GetBlockFinalityByNumber(blockNr rpc.BlockNumber) (*GetBlockFinalityResult, error) {
	return getBlockFinalityByHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithNumber(blockNr)))
}

// Return the validators that signed the block identified by hash,
// with the block and epoch context and the derivation inputs it was computed from.
// The signers are the creator and attestor of the given block plus unique senders of successful BlockSigner transactions for that block; all are restricted to the epoch's validator set.
func (api *PublicPosvDebugAPI) GetBlockSignersByHash(hash common.Hash) (*GetBlockSignersResult, error) {
	return getBlockSignersByHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithHash(hash, false)))
}

// Return the validators that signed the block identified by either a block hash or a block number,
// with the block and epoch context and the derivation inputs it was computed from.
// The signers are the creator and attestor of the given block plus unique senders of successful BlockSigner transactions for that block; all are restricted to the epoch's validator set.
func (api *PublicPosvDebugAPI) GetBlockSignersByHashOrNumber(blockNrOrHash rpc.BlockNumberOrHash) (*GetBlockSignersResult, error) {
	return getBlockSignersByHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, blockNrOrHash))
}

// Return the validators that signed the block at the given number,
// with the block and epoch context and the derivation inputs it was computed from.
// The signers are the creator and attestor of the given block plus unique senders of successful BlockSigner transactions for that block; all are restricted to the epoch's validator set.
func (api *PublicPosvDebugAPI) GetBlockSignersByNumber(blockNr rpc.BlockNumber) (*GetBlockSignersResult, error) {
	return getBlockSignersByHeaderWithDebugInfo(api.e, headerByNumberOrHash(api.e, rpc.BlockNumberOrHashWithNumber(blockNr)))
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

// Scan forward from targetNumber to the current chain head for the block with the highest signer finality.
// When the TIPSigning fork is active, only boundary blocks are checked, otherwise every block is checked.
// Return the best header found and its finality percentage, or nil when no candidate could be evaluated.
func findClosestFinalizedBlock(e *Ethereum, engine *posv.Posv, targetNumber uint64) (*types.Header, uint) {
	cfg := e.blockchain.Config()
	headNumber := e.blockchain.CurrentBlock().NumberU64()

	step := uint64(1)
	scanStart := targetNumber
	if cfg.IsTIPSigning(new(big.Int).SetUint64(headNumber)) {
		step = cfg.Viction.ValidatorSignInterval
		if rem := targetNumber % step; rem != 0 {
			scanStart = targetNumber + (step - rem)
		}
	}

	var (
		bestHeader   *types.Header
		bestFinality uint
	)
	for number := scanStart; number <= headNumber; number += step {
		header := e.blockchain.GetHeaderByNumber(number)
		if header == nil {
			continue
		}
		finality, err := getSignerFinalityOfHeader(e, header, engine)
		if err != nil {
			continue
		}
		if finality > bestFinality {
			bestHeader = header
			bestFinality = finality
		}
		if finality == 100 {
			break
		}
	}
	return bestHeader, bestFinality
}

// Return the header of the block whose BlockSigner transactions record the given header:
//   - Under TIPSigning, signing is aggregated at ValidatorSignInterval boundaries, so the next boundary block is used.
//   - Otherwise the given block itself.
//   - Return nil when no such block exists, e.g. for the genesis block.
func findNearestSignedBlock(e *Ethereum, header *types.Header) *types.Header {
	if header.Number.Int64() <= 0 {
		return nil
	}
	cfg := e.blockchain.Config()
	blockNumber := header.Number.Uint64()
	signedBlockNumber := blockNumber + (cfg.Viction.ValidatorSignInterval - (blockNumber % cfg.Viction.ValidatorSignInterval))
	latestBlockNumber := e.blockchain.CurrentBlock().NumberU64()
	if signedBlockNumber >= latestBlockNumber || !cfg.IsTIPSigning(header.Number) {
		signedBlockNumber = blockNumber
	}

	// The epoch checkpoint of the signed block must exist on the canonical chain.
	checkpointNumber := signedBlockNumber - (signedBlockNumber % cfg.Posv.Epoch)
	if e.blockchain.GetHeaderByNumber(checkpointNumber) == nil {
		return nil
	}
	return e.blockchain.GetHeaderByNumber(signedBlockNumber)
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

// Compute the finality percentage of the block at the given header, together with the signer share and the nearest finalized block ahead it was reconciled with.
// Genesis is final by definition. A block whose signer share cannot be computed still gets the finality of the nearest finalized block ahead when that block is on the same path.
func getBlockFinalityByHeader(e *Ethereum, header *types.Header) (uint, uint, *types.Header, uint, error) {
	if header == nil {
		return 0, 0, nil, 0, errors.New("header at block number or hash is not found")
	}
	cfg := e.blockchain.Config()
	if cfg.Posv == nil || cfg.Viction == nil {
		return 0, 0, nil, 0, core.ErrPosvRequired
	}
	engine, ok := e.engine.(*posv.Posv)
	if !ok {
		return 0, 0, nil, 0, errors.New("engine is not a posv engine")
	}
	if header.Number.Uint64() == 0 {
		return 100, 100, nil, 0, nil
	}

	signerFinality, err := getSignerFinalityOfHeader(e, header, engine)
	if err != nil {
		// The signer share failed; the forward scan below still produces a finality.
		log.Error("[PosvAPI] Failed to compute signer finality", "number", header.Number.Uint64(), "hash", header.Hash(), "err", err)
	}
	if signerFinality == 100 {
		return 100, 100, nil, 0, nil
	}

	closestHeader, closestFinality := findClosestFinalizedBlock(e, engine, header.Number.Uint64())
	if closestHeader != nil && closestFinality > signerFinality &&
		e.blockchain.AreTwoBlockSamePath(closestHeader.Hash(), header.Hash()) {
		return closestFinality, signerFinality, closestHeader, closestFinality, nil
	}
	return signerFinality, signerFinality, closestHeader, closestFinality, nil
}

// Compute the block finality for a header and wrap it with the block and epoch context and the reconciliation steps for the posvdebug API.
func getBlockFinalityByHeaderWithDebugInfo(e *Ethereum, header *types.Header) (*GetBlockFinalityResult, error) {
	finality, signerFinality, closestHeader, closestFinality, err := getBlockFinalityByHeader(e, header)
	if err != nil {
		return nil, err
	}
	cfg := e.blockchain.Config()
	result := &GetBlockFinalityResult{
		BlockNumber:    header.Number.Uint64(),
		BlockHash:      header.Hash().Hex(),
		Epoch:          header.Number.Uint64() / cfg.Posv.Epoch,
		EpochSize:      cfg.Posv.Epoch,
		Finality:       finality,
		SignerFinality: signerFinality,
	}
	if closestHeader != nil {
		result.ClosestFinalizedNumber = closestHeader.Number.Uint64()
		result.ClosestFinalizedHash = closestHeader.Hash().Hex()
		result.ClosestFinalizedFinality = closestFinality
	}
	return result, nil
}

// Compute the validators that signed the block at the given header, together with the signed block header and the validator set they were filtered against.
// Blocks without a signed block, such as the genesis block, return an empty list.
func getBlockSignersByHeader(e *Ethereum, header *types.Header) ([]common.Address, *types.Header, []common.Address, error) {
	if header == nil {
		return nil, nil, nil, errors.New("header at block number or hash is not found")
	}
	cfg := e.blockchain.Config()
	if cfg.Posv == nil || cfg.Viction == nil {
		return nil, nil, nil, core.ErrPosvRequired
	}
	engine, ok := e.engine.(*posv.Posv)
	if !ok {
		return nil, nil, nil, errors.New("engine is not a posv engine")
	}
	signedHeader := findNearestSignedBlock(e, header)
	if signedHeader == nil {
		return []common.Address{}, nil, nil, nil
	}
	signers, validators, err := getSigners(e, signedHeader, engine)
	if err != nil {
		return nil, nil, nil, err
	}
	return signers, signedHeader, validators, nil
}

// Compute the block signers for a header and wrap them with the block and epoch context for the debugging.
func getBlockSignersByHeaderWithDebugInfo(e *Ethereum, header *types.Header) (*GetBlockSignersResult, error) {
	signers, signedHeader, validators, err := getBlockSignersByHeader(e, header)
	if err != nil {
		return nil, err
	}
	cfg := e.blockchain.Config()
	result := &GetBlockSignersResult{
		BlockNumber: header.Number.Uint64(),
		BlockHash:   header.Hash().Hex(),
		Epoch:       header.Number.Uint64() / cfg.Posv.Epoch,
		EpochSize:   cfg.Posv.Epoch,
		Validators:  validators,
		Signers:     signers,
	}
	if signedHeader != nil {
		result.SignedBlockNumber = signedHeader.Number.Uint64()
		result.SignedBlockHash = signedHeader.Hash().Hex()
	}
	return result, nil
}

// Return the validator set governing the block: the validators recorded in the checkpoint header of the block's signing window.
// The window ends at the next ValidatorSignInterval boundary ahead of the block, or at the block itself when the boundary is not yet in the chain or the TIP2019 fork is inactive.
func getGoverningValidators(e *Ethereum, header *types.Header) []common.Address {
	cfg := e.blockchain.Config()
	interval := cfg.Viction.ValidatorSignInterval
	number := header.Number.Uint64()

	windowEnd := number + (interval - (number % interval))
	if windowEnd >= e.blockchain.CurrentBlock().NumberU64() || !cfg.IsTIP2019(header.Number) {
		windowEnd = number
	}
	checkpointNumber := windowEnd - (windowEnd % cfg.Posv.Epoch)
	checkpointHeader := e.blockchain.GetHeaderByNumber(checkpointNumber)
	if checkpointHeader == nil {
		return nil
	}
	return posv.ExtractValidatorsFromCheckpointHeader(checkpointHeader)
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

// Compute the signer finality of the block at the given header: the share of its governing validator set that signed the block, as recorded by the BlockSigner transactions of its signed block.
// Blocks without a signed block, such as the genesis block, blocks whose governing validator set is empty and blocks off the canonical path of their signed block have zero signer finality.
func getSignerFinalityOfHeader(e *Ethereum, header *types.Header, engine *posv.Posv) (uint, error) {
	validators := getGoverningValidators(e, header)
	if len(validators) == 0 {
		return 0, nil
	}
	signedHeader := findNearestSignedBlock(e, header)
	if signedHeader == nil {
		return 0, nil
	}
	if !e.blockchain.AreTwoBlockSamePath(signedHeader.Hash(), header.Hash()) {
		return 0, nil
	}
	signers, _, err := getSigners(e, signedHeader, engine)
	if err != nil {
		return 0, err
	}
	return uint(100 * len(signers) / len(validators)), nil
}

// Return the validators that signed the given signed block, together with the validator set the signers were filtered against.
// The signers are the creator and attestor of the given block plus unique senders of successful BlockSigner transactions for that block; all are restricted to the epoch's validator set.
func getSigners(e *Ethereum, signedHeader *types.Header, engine *posv.Posv) ([]common.Address, []common.Address, error) {
	cfg := e.blockchain.Config()
	number := signedHeader.Number.Uint64()
	checkpointNumber := number - (number % cfg.Posv.Epoch)
	checkpointHeader := e.blockchain.GetHeaderByNumber(checkpointNumber)
	if checkpointHeader == nil {
		return nil, nil, fmt.Errorf("checkpoint header #%d not found", checkpointNumber)
	}
	validators := posv.ExtractValidatorsFromCheckpointHeader(checkpointHeader)
	if len(validators) == 0 {
		return nil, nil, fmt.Errorf("no validators at checkpoint #%d", checkpointNumber)
	}

	signers, err := getSignersFromBlocks(e, number, signedHeader.Hash(), validators)
	if err != nil {
		return nil, nil, err
	}
	// A failed signature recovery yields no address the validator-set filter below would keep, so failed recoveries of the attestor and creator are skipped.
	if attestor, err := engine.Attestor(signedHeader); err == nil {
		signers = append(signers, attestor)
	}
	if creator, err := engine.Author(signedHeader); err == nil {
		signers = append(signers, creator)
	}

	filtered := make([]common.Address, 0, len(validators))
	for _, validator := range validators {
		for _, signer := range signers {
			if signer == validator {
				filtered = append(filtered, validator)
				break
			}
		}
	}
	return filtered, validators, nil
}

// Return the unique senders of successful BlockSigner transactions for the given block hash, scanned from the blocks following it up to the finality limit.
func getSignersFromBlocks(e *Ethereum, blockNumber uint64, blockHash common.Hash, validators []common.Address) ([]common.Address, error) {
	cfg := e.blockchain.Config()
	addrs := []common.Address{}
	isValidator := map[common.Address]bool{}
	for _, validator := range validators {
		isValidator[validator] = true
	}
	signer := types.MakeSigner(cfg, new(big.Int).SetUint64(blockNumber))

	limitNumber := blockNumber + cfg.Viction.ConsensusLimitTimeFinality
	if headNumber := e.blockchain.CurrentBlock().NumberU64(); limitNumber > headNumber {
		limitNumber = headNumber
	}
	for i := blockNumber + 1; i <= limitNumber; i++ {
		header := e.blockchain.GetHeaderByNumber(i)
		if header == nil {
			continue
		}
		signTxs, err := victionapi.GetBlockSignData(header, cfg, cfg.Viction, e.blockchain, e.blockchain, e.blockSignersCache)
		if err != nil {
			return addrs, err
		}
		for _, signtx := range signTxs {
			blkHash := common.BytesToHash(signtx.Data()[len(signtx.Data())-32:])
			from, err := types.Sender(signer, &signtx)
			if err != nil {
				// A sender that does not recover is not a signer; skip its transaction.
				continue
			}
			if blkHash == blockHash && isValidator[from] {
				addrs = append(addrs, from)
				delete(isValidator, from)
			}
		}
		if len(isValidator) == 0 {
			break
		}
	}
	return addrs, nil
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
