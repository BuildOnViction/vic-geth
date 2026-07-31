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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	lru "github.com/hashicorp/golang-lru"
)

const (
	attestorHeaderItemLength = 4
)

// EpochReward stores number of sign made by each validator and rewards for
// all stakeholders (validators and voters) in an epoch.
type EpochReward struct {
	Rewards            map[common.Address]map[common.Address]*big.Int `json:"rewards"`
	ValidatorRewards   map[common.Address]*ValidatorReward            `json:"signers"`
	StakeholderRewards map[common.Address]*big.Int                    `json:"-"`
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
	PosvGetBlockSignData(config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader) ([]types.Transaction, error)

	// Get creator-attestor pairs from the state.
	PosvGetCreatorAttestorPairs(c *Posv, config *params.ChainConfig, header, checkpointHeader *types.Header) (map[common.Address]common.Address, uint64, error)

	// Calculate and distribute reward at the end of each epoch.
	PosvGetEpochReward(c *Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
		header *types.Header, chain consensus.ChainReader, state *state.StateDB, logger log.Logger) (*EpochReward, error)

	// Add balance rewards to the state (apply the rewards returned by PosvGetEpochReward).
	PosvDistributeEpochRewards(header *types.Header, state *state.StateDB, epochReward *EpochReward) error

	// Penalize validators for creating bad block or not creating block at all.
	PosvGetPenalties(c *Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader, validators []common.Address) ([]common.Address, error)

	// Get eligble validators from the state.
	PosvGetValidators(vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader) ([]common.Address, error)
}

// Return Ethereum address recovered from the signature in header's Attestor field.
func (c *Posv) Attestor(header *types.Header) (common.Address, error) {
	return ecrecover2(header, c.attestSignatures)
}

// Get all BlockSign transactions for a given block. If it's not cached yet, get it from the state.
func (c *Posv) GetSignDataForBlock(config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader) ([]types.Transaction, error) {
	if header == nil {
		return nil, fmt.Errorf("GetSignDataForBlock: header is nil")
	}
	blockHash := header.Hash()
	if signers, ok := c.BlockSigners.Get(blockHash); ok {
		if signers, ok := signers.([]types.Transaction); ok && signers != nil {
			return signers, nil
		}
	}
	signers, err := c.backend.PosvGetBlockSignData(config, vicConfig, header, chain)
	if err != nil {
		return nil, err
	}
	c.BlockSigners.Add(blockHash, signers)
	return signers, nil
}

// Check if the signer is inturn to mint current block. Also return context of the check including:
// currentIndex, parentIndex, validatorCount.
//
// If the parent's sealer (Author(parent)) is not present in validators (checkpoint extra order),
// parentIndex stays -1 and in-turn is only the signer at validators[0] because (-1+1)%n == 0.
// If your etherbase is not validators[0], IsMyTurn is false for every other validator until the
// checkpoint list matches the chain. Align genesis/checkpoint extra with actual signers.
func (c *Posv) IsMyTurn(signer common.Address, parent *types.Header, validators []common.Address) (bool, int, int, int, error) {
	validatorsCount := len(validators)
	if validatorsCount == 0 {
		return false, -1, -1, 0, errEmptyValidators
	}

	parentIndex := -1
	if parent.Number.Uint64() > 0 {
		parentCreator, err := c.Author(parent)
		if err != nil {
			return false, 0, 0, 0, err
		}
		parentIndex = common.IndexOf(validators, parentCreator)
	}
	currentIndex := common.IndexOf(validators, signer)

	inturn := (parentIndex+1)%validatorsCount == currentIndex
	return inturn, currentIndex, parentIndex, validatorsCount, nil
}

// GetCheckpointHeader returns the checkpoint header for the epoch containing
// the given header. If the header itself is a checkpoint (number % epoch == 0)
// it is returned directly. Otherwise it tries the canonical DB first (prior
// epochs already committed), then falls back to the in-memory recentHeaders
// cache which is populated by verifyHeaderWithCache as each checkpoint in the
// current batch is successfully verified.
func GetCheckpointHeader(posvConfig *params.PosvConfig, header *types.Header, chain consensus.ChainHeaderReader, parents []*types.Header) *types.Header {
	blockNumber := header.Number.Uint64()
	if blockNumber%posvConfig.Epoch == 0 {
		return header
	}
	prevCheckpointBlockNumber := blockNumber - (blockNumber % posvConfig.Epoch)

	// Try canonical DB first (covers prior epochs already committed).
	if h := chain.GetHeaderByNumber(prevCheckpointBlockNumber); h != nil {
		return h
	}
	for _, parent := range parents {
		if parent.Number.Uint64() == prevCheckpointBlockNumber {
			return parent
		}
	}

	return nil
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
	addressLengthInt := int(common.AddressLength)
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

	validators := make([]common.Address, (len(header.Extra)-ExtraVanity-ExtraSeal)/int(common.AddressLength))
	for i := 0; i < len(validators); i++ {
		copy(validators[i][:], header.Extra[ExtraVanity+i*int(common.AddressLength):])
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

// Return the distance between current index and parent index in the circular list of validators.
func distance(currentIndex, parentIndex, validatorCount int) int {
	if currentIndex > parentIndex {
		return currentIndex - parentIndex
	}
	return validatorCount + currentIndex - parentIndex
}

// ecrecover2 extracts the Ethereum account address from a Attestor header.
func ecrecover2(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()

	// hitrate while straight-forward sync is from 0.5 to 0.65
	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}

	// Retrieve the signature from the header extra-data
	if len(header.Attestor) != ExtraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Attestor

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(SealHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}

	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}
