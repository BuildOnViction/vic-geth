// Copyright 2025 The Viction Authors
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

package viction

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// Return list of validators which don't create block follow the consensus rule after TIPSigning
func PenalizeValidatorsTIPSigning(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader, logger log.Logger,
) ([]common.Address, error) {
	blockNumber := header.Number.Uint64()
	blockNumberBig := header.Number
	prevCheckpointBlockNumber := blockNumber - posvConfig.Epoch
	penalties := []common.Address{}

	// First epoch doesn't have penalty
	if prevCheckpointBlockNumber <= 0 {
		return penalties, nil
	}

	// Count number of blocks mined by each validator
	epochBlockHashes := make([]common.Hash, posvConfig.Epoch)
	blockMiningCounts := map[common.Address]uint64{}
	blockHash := header.ParentHash
	for i := uint64(0); i < posvConfig.Epoch; i++ {
		epochBlockHashes[i] = blockHash
		headr := chain.GetHeaderByHash(blockHash)
		miner, _ := c.Author(headr)
		if count, ok := blockMiningCounts[miner]; ok {
			blockMiningCounts[miner] = count + 1
		} else {
			blockMiningCounts[miner] = 1
		}
		blockHash = header.ParentHash
	}

	// Penalize validators didn't create block or lower than required
	prevCheckpointHeader := chain.GetHeaderByNumber(prevCheckpointBlockNumber)
	validators := posv.ExtractValidatorsFromCheckpointHeader(prevCheckpointHeader)
	for _, validator := range validators {
		if _, exist := blockMiningCounts[validator]; !exist {
			penalties = append(penalties, validator)
		}
	}
	for miner, count := range blockMiningCounts {
		if count < vicConfig.ValidatorMinBlockPerEpochCount {
			penalties = append(penalties, miner)
		}
	}

	// Get list of previously penalized validators for BlockSign check
	comebackCheckpointBlockNumber := uint64(0)
	comebackLength := (vicConfig.PenaltyEpochCount + 1) * posvConfig.Epoch
	if blockNumber > comebackLength {
		comebackCheckpointBlockNumber = blockNumber - comebackLength
	}
	comebacks := []common.Address{}
	if comebackCheckpointBlockNumber > 0 {
		combackHeader := chain.GetHeaderByNumber(comebackCheckpointBlockNumber)
		penalties := posv.DecodePenaltiesFromHeader(combackHeader.Penalties)
		for _, p := range penalties {
			for _, addr := range validators {
				if p == addr {
					comebacks = append(comebacks, p)
				}
			}
		}
	}

	// If penalized validators has BlockSign recently, remove them from penalties
	if len(comebacks) > 0 {
		mapBlockHash := map[common.Hash]bool{}
		for i := vicConfig.PenaltyComebackBlockCount - 1; i >= 0; i-- {
			blockNumber := header.Number.Uint64() - i - 1
			headr := chain.GetHeaderByNumber(blockNumber)
			blockHash := epochBlockHashes[i]
			if blockNumber%vicConfig.ValidatorSignInterval == 0 {
				mapBlockHash[blockHash] = true
			}
			txs := c.GetSignDataForBlock(config, vicConfig, headr, chain, logger)
			// Check for BlockSign of specific signer
			for _, tx := range txs {
				signedBlockHash := common.BytesToHash(tx.Data()[len(tx.Data())-32:])
				signer, _ := tx.Sender()
				if mapBlockHash[signedBlockHash] {
					for j, addr := range comebacks {
						if signer == addr {
							comebacks = append(comebacks[:j], comebacks[j+1:]...)
							break
						}
					}
				}
			}
		}
	}

	penalties = append(penalties, comebacks...)
	if config.IsTIPRandomize(blockNumberBig) {
		return penalties, nil
	}
	return comebacks, nil
}

// Return list of validators which don't create block follow the consensus rule.
func PenalizeValidatorsDefault(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader, logger log.Logger,
) ([]common.Address, error) {
	blockNumber := header.Number.Uint64()
	prevCheckpointBlockNumber := blockNumber - posvConfig.Epoch
	penalties := []common.Address{}

	// First epoch doesn't have penalty
	if prevCheckpointBlockNumber <= 0 {
		return penalties, nil
	}

	prevCheckpointHeader := chain.GetHeaderByNumber(prevCheckpointBlockNumber)
	validators := posv.ExtractValidatorsFromCheckpointHeader(prevCheckpointHeader)
	if len(validators) == 0 {
		return penalties, nil
	}

	for i := prevCheckpointBlockNumber; i < blockNumber; i++ {
		iBig := new(big.Int).SetUint64(i)
		if i%vicConfig.ValidatorSignInterval == 0 || !config.IsTIP2019(iBig) {
			headr := chain.GetHeaderByNumber(i)
			if len(validators) == 0 {
				break
			}
			txs := c.GetSignDataForBlock(config, vicConfig, headr, chain, logger)
			// Check for BlockSign of specific signer
			for _, tx := range txs {
				signer, _ := tx.Sender()
				for j, addr := range validators {
					if signer == addr {
						validators = append(validators[:j], validators[j+1:]...)
					}
				}
			}
		}
	}
	return validators, nil
}
