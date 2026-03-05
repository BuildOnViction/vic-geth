package viction

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func GetCreatorAttestorPairs(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig,
	header, checkpointHeader *types.Header,
) (map[common.Address]common.Address, uint64, error) {
	number := header.Number.Uint64()
	validators := posv.ExtractValidatorsFromCheckpointHeader(checkpointHeader)
	attestorIdxs := posv.ExtractAttestorsFromCheckpointHeader(checkpointHeader)
	return getCreatorAttestorPairs(config, posvConfig, number, validators, attestorIdxs)
}

func getCreatorAttestorPairs(config *params.ChainConfig, posvConfig *params.PosvConfig,
	number uint64, validators []common.Address, attestorIdxs []int64,
) (map[common.Address]common.Address, uint64, error) {
	results := map[common.Address]common.Address{}
	validatorCount := uint64(len(validators))
	attestorCount := uint64(len(attestorIdxs))
	offset := uint64(0)
	if validatorCount > attestorCount {
		return nil, offset, ErrInvalidAttestorList
	}
	if validatorCount > 0 {
		if config.IsTIPRandomize(new(big.Int).SetUint64(number)) {
			offset = ((number % posvConfig.Epoch) / validatorCount) % validatorCount
		}
		for i, val := range validators {
			attIdx := uint64(attestorIdxs[i]) % validatorCount
			attIdx = (attIdx + offset) % validatorCount
			results[val] = validators[attIdx]
		}
	}
	return results, offset, nil
}
