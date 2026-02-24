package viction

import (
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/contracts/validator/contract"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/tforce-io/tf-golib/stdx/mathxt/bigxt"
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

// Get eligble validators from the state.
//
// *NOTE: The injected state must be at the checkpoint block.
func GetValidators(vicConfig *params.VictionConfig, state *state.StateDB, client bind.ContractBackend,
) ([]common.Address, error) {
	validatorContract, _ := contract.NewValidator(vicConfig.ValidatorContract, client)

	opts := new(bind.CallOpts)
	addresses, err := validatorContract.GetCandidates(opts)
	if err != nil {
		return nil, err
	}

	candidates := []*posv.ValidatorInfo{}
	for _, addr := range addresses {
		cap, err := validatorContract.GetCandidateCap(opts, addr)
		if err != nil {
			return nil, err
		}
		if addr == (common.Address{}) {
			continue
		}
		candidates = append(candidates, &posv.ValidatorInfo{Address: addr, Capacity: cap})
	}
	sort.Slice(candidates, func(i, j int) bool {
		return bigxt.IsGreaterThanOrEqualInt(candidates[i].Capacity, candidates[j].Capacity)
	})
	validatorMaxCountInt := int(vicConfig.ValidatorMaxCount)
	if len(candidates) > validatorMaxCountInt {
		candidates = candidates[:validatorMaxCountInt]
	}
	validators := []common.Address{}
	for _, candidate := range candidates {
		validators = append(validators, candidate.Address)
	}
	return validators, nil
}
