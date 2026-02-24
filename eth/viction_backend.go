package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/viction"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/tforce-io/tf-golib/stdx/mathxt/bigxt"
)

const SignMethodHex = "e341eaa4"

// Get attestors from list of validators at checkpoint block.
func (s *Ethereum) PosvGetAttestors(vicConfig *params.VictionConfig, header *types.Header, validators []common.Address,
) ([]int64, error) {
	rpcClient, err := s.GetIPCClient()
	if err != nil {
		return nil, err
	}
	client := ethclient.NewClient(rpcClient)
	return viction.GetAttestors(vicConfig, validators, client)
}

// Get block signers from the state.
func (s *Ethereum) PosvGetBlockSignData(config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader,
) []types.Transaction {
	blockNumber := header.Number
	block := chain.GetBlock(header.Hash(), blockNumber.Uint64())
	data := []types.Transaction{}
	transactions := block.Transactions()
	if config.IsTIPSigning(blockNumber) {
		for _, tx := range transactions {
			if IsVicBlockSingingTx(*tx, vicConfig) {
				data = append(data, *tx)
			}
		}
	} else {
		// TODO: Handle receipts later
		for _, tx := range transactions {
			if IsVicBlockSingingTx(*tx, vicConfig) {
				data = append(data, *tx)
			}
		}
	}
	return data
}

// Get creator-attestor pairs from the state.
func (s *Ethereum) PosvGetCreatorAttestorPairs(c *posv.Posv, config *params.ChainConfig,
	header, checkpointHeader *types.Header,
) (map[common.Address]common.Address, uint64, error) {
	return viction.GetCreatorAttestorPairs(c, config, config.Posv, header, checkpointHeader)
}

// Calculate and distribute reward at checkpoint block.
func (s *Ethereum) PosvGetEpochReward(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader, state *state.StateDB, logger log.Logger,
) (*posv.EpochReward, error) {
	epochRewards := &posv.EpochReward{}
	blockNumber := header.Number.Uint64()
	blockNumberBig := header.Number

	if bigxt.IsLessThanOrEqualInt(blockNumberBig, new(big.Int).SetUint64(posvConfig.Epoch)) {
		return epochRewards, nil
	}

	// Get initial reward
	totalReward := viction.CalcDefaultRewardPerBlock((*big.Int)(vicConfig.RewardPerEpoch), blockNumber, posvConfig.BlocksPerYear())
	// Get additional reward for Saigon upgrade
	if chain.Config().IsSaigon(blockNumberBig) {
		saigonReward := viction.CalcSaigonRewardPerBlock((*big.Int)(vicConfig.SaigonRewardPerEpoch), chain.Config().SaigonBlock, blockNumber, posvConfig.BlocksPerYear())
		totalReward = new(big.Int).Add(totalReward, saigonReward)
	}

	// Calculate rewards for validators and stakeholders
	validatorRewards, _ := viction.CalcRewardsForValidators(c, config, posvConfig, vicConfig, header, totalReward, chain, logger)
	epochRewards.ValidatorRewards = validatorRewards

	stakeholderRewards, _ := viction.CalcRewardsForStakeholders(c, config, posvConfig, vicConfig, header, validatorRewards, state, logger)
	epochRewards.StakholderRewards = stakeholderRewards

	return epochRewards, nil
}

// Get list of validators creating bad block or not creating block at all.
func (s *Ethereum) PosvGetPenalties(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader,
) ([]common.Address, error) {
	if config.IsTIPSigning(header.Number) {
		return viction.PenalizeValidatorsTIPSigning(c, config, posvConfig, vicConfig, header, chain)
	}
	return viction.PenalizeValidatorsDefault(c, config, posvConfig, vicConfig, header, chain)
}

// Check a transaction is Viction BlockSign transaction.
func IsVicBlockSingingTx(tx types.Transaction, vicConfig *params.VictionConfig) bool {
	toAddr := tx.To()
	if toAddr == nil || *toAddr != vicConfig.ValidatorBlockSignContract {
		return false
	}

	data := tx.Data()
	method := common.Bytes2Hex(data[0:4])

	if method != SignMethodHex && len(data) >= 68 {
		return false
	}

	return true
}

// Get eligble validators from the state.
func (s *Ethereum) PosvGetValidators(vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader,
) ([]common.Address, error) {
	rpcClient, err := s.GetIPCClient()
	if err != nil {
		return nil, err
	}
	client := ethclient.NewClient(rpcClient)
	state, err := s.BlockChain().State()
	if err != nil {
		return nil, err
	}
	return viction.GetValidators(vicConfig, state, client)
}
