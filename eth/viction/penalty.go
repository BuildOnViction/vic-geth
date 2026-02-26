package viction

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func PenalizeValidatorsDefault(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader,
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
		if i%vicConfig.ValidatorSignInterval == 0 || !config.IsTIP2019(big.NewInt(int64(i))) {
			headr := chain.GetHeaderByNumber(i)
			if len(validators) == 0 {
				break
			}
			txs := c.GetSignDataForBlock(config, vicConfig, headr, chain)
			signer := types.MakeSigner(config, big.NewInt(int64(i)))
			// Check for BlockSign of specific signer
			for _, tx := range txs {
				from, err := types.Sender(signer, &tx)
				if err != nil {
					return nil, err
				}
				for j, addr := range validators {
					if from == addr {
						validators = append(validators[:j], validators[j+1:]...)
					}
				}
			}
		}
	}
	return validators, nil
}

// [TO-DO]
func PenalizeValidatorsTIPSigning(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader,
) ([]common.Address, error) {
	blockNumber := header.Number.Uint64()
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
			txs := c.GetSignDataForBlock(config, vicConfig, headr, chain)
			signer := types.MakeSigner(config, big.NewInt(int64(i)))
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
		}
	}

	penalties = append(penalties, comebacks...)
	if config.IsTIPRandomize(big.NewInt(int64(blockNumber))) {
		return penalties, nil
	}
	return comebacks, nil
}
