// Copyright 2016 The go-ethereum Authors
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

// Package posv implements the proof-of-stake-voting consensus engine.
package posv

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	attestorHeaderItemLength = 4
)

// EpochReward stores number of sign made by each validator and rewards for
// all stakeholders (validators and voters) in an epoch.
type EpochReward struct {
	ValidatorRewards  map[common.Address]*ValidatorReward `json:"signers"`
	StakholderRewards map[common.Address]*big.Int         `json:"rewards"`
}

type ValidatorReward struct {
	Sign   uint64   `json:"sign"`
	Reward *big.Int `json:"reward"`
}

// ValidatorInfo stores basic information about a validator.
type ValidatorInfo struct {
	Address  common.Address `json:"address"`
	Capacity *big.Int       `json:"capacity"`
	Owner    common.Address `json:"owner"`
}

type PosvBackend interface {
	// Get attestors from list of validators.
	PosvGetAttestors(vicConfig *params.VictionConfig, header *types.Header, validators []common.Address) ([]int64, error)

	// Get block signers from the state.
	PosvGetBlockSignData(config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader) []types.Transaction

	// Get creator-attestor pairs from the state.
	PosvGetCreatorAttestorPairs(c *Posv, config *params.ChainConfig, header, checkpointHeader *types.Header) (map[common.Address]common.Address, uint64, error)

	// Calculate and distribute reward at the end of each epoch.
	PosvGetEpochReward(c *Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
		header *types.Header, chain consensus.ChainReader, state *state.StateDB, logger log.Logger) (*EpochReward, error)

	// Add balance rewards to the state (apply the rewards returned by PosvGetEpochReward).
	PosvDistributeEpochRewards(header *types.Header, state *state.StateDB, epochReward *EpochReward) error

	// Penalize validators for creating bad block or not creating block at all.
	PosvGetPenalties(c *Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader) ([]common.Address, error)

	// Get eligble validators from the state.
	PosvGetValidators(vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader) ([]common.Address, error)
}

// Check If the given block is a checkpoint block, return it, else return previous checkpoint block header.
func GetCheckpointHeader(posvConfig *params.PosvConfig, header *types.Header, chain consensus.ChainHeaderReader) *types.Header {
	blockNumber := header.Number.Uint64()
	if blockNumber%posvConfig.Epoch == 0 {
		return header
	}
	prevCheckpointBlockNumber := blockNumber - (blockNumber % posvConfig.Epoch)
	prevCheckpointHeader := chain.GetHeaderByNumber(prevCheckpointBlockNumber)
	return prevCheckpointHeader
}

// Encode list of attestor numbers into bytes following format of Block.Attestors.
func EncodeAttestorsForHeader(attestors []int64) []byte {
	var attestorsBuff []byte
	for _, attestor := range attestors {
		attestorBuff := common.LeftPadBytes([]byte(fmt.Sprintf("%d", attestor)), attestorHeaderItemLength)
		attestorsBuff = append(attestorsBuff, attestorBuff...)
	}
	return attestorsBuff
}

// Encode list of penalized addresses into bytes following format of Block.Penalties.
func EncodePenaltiesForHeader(penalties []common.Address) []byte {
	var penaltiesBuff []byte
	for _, attestor := range penalties {
		penaltiesBuff = append(penaltiesBuff, attestor.Bytes()...)
	}
	return penaltiesBuff
}

// Decode bytes with format of Block.Penalties into list of addresses.
func DecodePenaltiesFromHeader(penaltiesBuff []byte) []common.Address {
	addressLengthInt := int(AddressLength)
	penaltyCount := len(penaltiesBuff) / addressLengthInt
	penalties := make([]common.Address, penaltyCount)
	for i := 0; i < penaltyCount; i++ {
		penaltyBuff := penaltiesBuff[i*addressLengthInt : (i+1)*addressLengthInt]
		penalties[i] = common.BytesToAddress(penaltyBuff)
	}
	return penalties
}

// Process block header Extra field of a checkpoint block to return the list of new validators.
func ExtractValidatorsFromCheckpointHeader(header *types.Header) []common.Address {
	if header == nil {
		return []common.Address{}
	}

	validators := make([]common.Address, (len(header.Extra)-ExtraVanity-ExtraSeal)/int(AddressLength))
	for i := 0; i < len(validators); i++ {
		copy(validators[i][:], header.Extra[ExtraVanity+i*int(AddressLength):])
	}

	return validators
}

// Process block header NewAttestors field of a checkpoint block to return the list of new attestors.
func ExtractAttestorsFromCheckpointHeader(header *types.Header) []int64 {
	if header == nil {
		return []int64{}
	}

	attestors := DecodeAttestorsFromHeader(header.NewAttestors)
	return attestors
}

// Decode bytes with format of Block.Attestors into list of attestor numbers.
func DecodeAttestorsFromHeader(attestorsBuff []byte) []int64 {
	attestorCount := len(attestorsBuff) / attestorHeaderItemLength
	attestors := make([]int64, attestorCount)
	for i := 0; i < attestorCount; i++ {
		attestorBuff := bytes.Trim(attestorsBuff[i*attestorHeaderItemLength:(i+1)*attestorHeaderItemLength], "\x00")
		attestorNumber, err := strconv.ParseInt(string(attestorBuff), 10, 64)
		if err != nil {
			return []int64{}
		}
		attestors[i] = attestorNumber
	}

	return attestors
}

// Get all BlockSign transactions for a given block. If it's not cached yet, get it from the state.
func (c *Posv) GetSignDataForBlock(config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader) []types.Transaction {
	blockHash := header.Hash()
	if signers, ok := c.BlockSigners.Get(blockHash); ok {
		return signers.([]types.Transaction)
	}
	signers := c.backend.PosvGetBlockSignData(config, vicConfig, header, chain)
	c.BlockSigners.Add(blockHash, signers)
	return signers
}
