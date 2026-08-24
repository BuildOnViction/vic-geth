package viction

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

func PenalizeValidatorsDefault(bc *core.BlockChain, c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader,
) ([]common.Address, error) {
	if bc == nil {
		return []common.Address{}, fmt.Errorf("blockchain not initialized (block %v)", header.Number)
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

func PenalizeValidatorsTIPSigning(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader,
	validators []common.Address,
) ([]common.Address, error) {
	if header == nil {
		return nil, fmt.Errorf("penalize TIP signing: checkpoint header is nil")
	}
	blockNumber := header.Number.Uint64()
	log.Debug("PenalizeValidatorsTIPSigning", "checkpoint", blockNumber, "hash", header.Hash())
	prevCheckpointBlockNumber := blockNumber - posvConfig.Epoch
	penalties := []common.Address{}

	// First epoch doesn't have penalty
	if prevCheckpointBlockNumber <= 0 {
		return penalties, nil
	}

	// Count number of blocks mined by each validator
	epochBlockHashes := make([]common.Hash, posvConfig.Epoch)
	blockMiningCounts := map[common.Address]uint64{}
	epochBlockHashes[0] = header.ParentHash
	parentHash := header.ParentHash
	for i := uint64(1); i < posvConfig.Epoch; i++ {
		parentHeader := chain.GetHeaderByHash(parentHash)
		if parentHeader == nil {
			log.Warn("PenalizeValidatorsTIPSigning: parent header missing from chain reader",
				"checkpoint", blockNumber, "epochStep", i, "parentHash", parentHash.Hex())
			return nil, fmt.Errorf("penalize TIP signing: GetHeaderByHash returned nil (checkpoint %d, step %d, parentHash %s)",
				blockNumber, i, parentHash.Hex())
		}
		miner, err := c.Author(parentHeader)
		if err != nil {
			log.Warn("PenalizeValidatorsTIPSigning: Author failed",
				"checkpoint", blockNumber, "epochStep", i, "headerNumber", parentHeader.Number, "err", err)
			return nil, fmt.Errorf("penalize TIP signing: author at step %d: %w", i, err)
		}
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
		if combackHeader == nil {
			log.Warn("PenalizeValidatorsTIPSigning: comeback checkpoint header missing",
				"checkpoint", blockNumber, "comebackCheckpoint", comebackCheckpointBlockNumber)
			return nil, fmt.Errorf("penalize TIP signing: comeback header nil at block %d", comebackCheckpointBlockNumber)
		}
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
			hNum := header.Number.Uint64() - uint64(i) - 1
			h := chain.GetHeaderByNumber(hNum)
			blockHash := epochBlockHashes[i]
			if hNum%vicConfig.ValidatorSignInterval == 0 {
				mapBlockHash[blockHash] = true
			}
			if h == nil {
				log.Warn("PenalizeValidatorsTIPSigning: header missing for BlockSign scan",
					"checkpoint", blockNumber, "blockNumber", hNum)
				return nil, fmt.Errorf("penalize TIP signing: header nil at block %d", hNum)
			}
			txs, err := c.GetSignDataForBlock(config, vicConfig, h, chain)
			if err != nil {
				return []common.Address{}, err
			}
			signer := types.MakeSigner(config, big.NewInt(int64(hNum)))
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
