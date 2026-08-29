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

package viction

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	lru "github.com/hashicorp/golang-lru"
)

// Return addresses of penaltized validators following default rule based on current block number.
func PenalizeValidatorsDefault(
	config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader, bc *core.BlockChain,
) ([]common.Address, error) {
	if bc == nil {
		return []common.Address{}, fmt.Errorf("penalize/default: blockchain not initialized (block %v)", header.Number)
	}
	// Viction reads signers from the contract using the state trie at the checkpoint block.
	// This avoids relying on where the BlockSign tx ended up being included.
	statedb, err := bc.State()
	if err != nil {
		return nil, fmt.Errorf("penalize/default: failed to get statedb at checkpoint root: %w", err)
	}
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
		// Only check blocks that can be signed (sign interval) and/or pre-TIP blocks.
		if i%vicConfig.ValidatorSignInterval != 0 && config != nil && config.IsTIP2019(big.NewInt(int64(i))) {
			continue
		}

		h := chain.GetHeaderByNumber(i)
		if h == nil {
			continue
		}
		blk := bc.GetBlock(h.Hash(), i)
		if blk == nil {
			continue
		}

		signers := statedb.GetSigners(vicConfig.ValidatorBlockSignContract, blk)
		for _, signer := range signers {
			for j, addr := range validators {
				if signer == addr {
					validators = append(validators[:j], validators[j+1:]...)
				}
			}
		}
	}

	return validators, nil
}

// Return addresses of penaltized validators following TIPSigning rule based on current block number.
func PenalizeValidatorsTIPSigning(
	config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header, validators []common.Address,
	chain consensus.ChainReader, bc *core.BlockChain, blockSigners *lru.ARCCache,
) ([]common.Address, error) {
	blockNumber := header.Number.Uint64()
	prevCheckpointBlockNumber := blockNumber - posvConfig.Epoch
	penalties := []common.Address{}

	// First epoch doesn't have penalty
	if prevCheckpointBlockNumber <= 0 {
		return penalties, nil
	}

	// Count number of blocks mined by each validator
	sigCache, _ := lru.NewARC(int(posvConfig.Epoch))
	epochBlockHashes := make([]common.Hash, posvConfig.Epoch)
	blockMiningCounts := map[common.Address]uint64{}
	epochBlockHashes[0] = header.ParentHash
	parentHash := header.ParentHash
	for i := uint64(1); i < posvConfig.Epoch; i++ {
		parentHeader := chain.GetHeaderByHash(parentHash)
		miner, _ := posv.Ecrecover(parentHeader, sigCache)
		if count, ok := blockMiningCounts[miner]; ok {
			blockMiningCounts[miner] = count + 1
		} else {
			blockMiningCounts[miner] = 1
		}
		parentHash = parentHeader.ParentHash
		epochBlockHashes[i] = parentHash
	}

	// Penalize validators didn't create block or lower than required
	prevCheckpointHeader := chain.GetHeaderByNumber(prevCheckpointBlockNumber)
	preValidators := posv.ExtractValidatorsFromCheckpointHeader(prevCheckpointHeader)
	for _, validator := range preValidators {
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
	mapBlockHash := map[common.Hash]bool{}
	for i := int(vicConfig.PenaltyComebackBlockCount) - 1; i >= 0; i-- {
		if len(comebacks) > 0 {
			blockNumber := header.Number.Uint64() - uint64(i) - 1
			header := chain.GetHeaderByNumber(blockNumber)
			blockHash := epochBlockHashes[i]
			if blockNumber%vicConfig.ValidatorSignInterval == 0 {
				mapBlockHash[blockHash] = true
			}
			txs, err := GetBlockSignData(config, vicConfig, header, chain, bc, blockSigners)
			if err != nil {
				return []common.Address{}, err
			}
			signer := types.MakeSigner(config, big.NewInt(int64(blockNumber)))
			// Check for BlockSign of specific signer
			for _, tx := range txs {
				signedBlockHash := common.BytesToHash(tx.Data()[len(tx.Data())-32:])
				from, err := types.Sender(signer, &tx)
				if err != nil {
					return nil, err
				}
				if mapBlockHash[signedBlockHash] {
					for j, addr := range comebacks {
						if from == addr {
							comebacks = append(comebacks[:j], comebacks[j+1:]...)
							break
						}
					}
				}
			}
		} else {
			break
		}
	}

	penalties = append(penalties, comebacks...)
	if config.IsTIPRandomize(big.NewInt(int64(blockNumber))) {
		return penalties, nil
	}
	return comebacks, nil
}
