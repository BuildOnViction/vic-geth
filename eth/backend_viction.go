package eth

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/viction"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/legacy/lending"
	"github.com/ethereum/go-ethereum/legacy/trading"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/sortlgc"
)

const SignMethodHex = "e341eaa4"

// Get attestors from list of validators.
func (s *Ethereum) PosvGetAttestors(
	vicConfig *params.VictionConfig, header *types.Header, validators []common.Address,
) ([]int64, error) {
	state, err := s.BlockChain().State()
	if err != nil {
		return nil, err
	}
	return viction.GetAttestors(vicConfig, validators, state)
}

// Get block signers from the state.
func (s *Ethereum) PosvGetBlockSignData(
	config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader,
) ([]types.Transaction, error) {
	if header == nil {
		return nil, fmt.Errorf("PosvGetBlockSignData: header is nil")
	}
	blockHash := header.Hash()
	blockNumber := header.Number
	block := chain.GetBlock(blockHash, blockNumber.Uint64())
	if block == nil {
		return nil, fmt.Errorf("PosvGetBlockSignData: block body not found (number=%d hash=%s)", blockNumber, blockHash)
	}
	data := []types.Transaction{}

	// Block-sign txs are EVM-executed and may fail.
	// Only successful signing txs count toward rewards and penalties.
	//
	// On post-Byzantium receipt format, `Receipt.Status` is the correct source
	// of success/failure. Using `len(PostState)` is unreliable and can misclassify.
	var receipts types.Receipts
	if config != nil {
		receipts = s.blockchain.GetReceiptsByHash(blockHash)
	}
	txs := block.Transactions()
	if receipts != nil && len(receipts) != len(txs) {
		return nil, fmt.Errorf(
			"PosvGetBlockSignData: receipts/tx count mismatch (number=%d hash=%s txs=%d receipts=%d)",
			blockNumber.Uint64(), blockHash, len(txs), len(receipts),
		)
	}

	for i, tx := range txs {
		if !tx.IsSigningTransaction(vicConfig.ValidatorBlockSignContract) {
			continue
		}
		if receipts != nil && i < len(receipts) {
			r := receipts[i]
			var status uint64
			if len(r.PostState) > 0 {
				status = types.ReceiptStatusSuccessful
			} else {
				status = r.Status
			}
			if status == types.ReceiptStatusFailed {
				continue
			}
		}
		data = append(data, *tx)
	}
	return data, nil
}

// Get creator-attestor pairs from the state.
func (s *Ethereum) PosvGetCreatorAttestorPairs(
	config *params.ChainConfig, posvConfig *params.PosvConfig, victionConfig *params.VictionConfig, header, checkpointHeader *types.Header,
) (map[common.Address]common.Address, uint64, error) {
	pairs, offset, err := viction.GetCreatorAttestorPairs(config, config.Posv, header, checkpointHeader)
	if err != viction.ErrInvalidAttestorList {
		// Either success or a non-recoverable error — propagate as-is.
		return pairs, offset, err
	}

	// Fallback: checkpointHeader.NewAttestors is absent or inconsistent.
	// Re-derive attestor indices from the randomize contract state at the
	// checkpoint root.  This is exactly what the miner computed in Prepare(),
	// so all nodes reach the same result deterministically.
	if config.Viction == nil || checkpointHeader == nil {
		return nil, 0, err
	}
	validators := posv.ExtractValidatorsFromCheckpointHeader(checkpointHeader)
	if len(validators) == 0 {
		return nil, 0, err
	}
	stateAtCheckpoint, sErr := s.blockchain.StateAt(checkpointHeader.Root)
	if sErr != nil {
		log.Warn("PosvGetCreatorAttestorPairs: state fallback failed, cannot load checkpoint state",
			"checkpoint", checkpointHeader.Number, "err", sErr)
		return nil, 0, err
	}
	attestorIdxs, aErr := viction.GetAttestors(config.Viction, validators, stateAtCheckpoint)
	if aErr != nil {
		log.Warn("PosvGetCreatorAttestorPairs: state fallback failed, cannot compute attestors",
			"checkpoint", checkpointHeader.Number, "err", aErr)
		return nil, 0, err
	}
	log.Warn("PosvGetCreatorAttestorPairs: NewAttestors absent in checkpoint, using state-based fallback",
		"checkpoint", checkpointHeader.Number, "number", header.Number)
	return viction.BuildCreatorAttestorPairs(config, config.Posv, header.Number.Uint64(), validators, attestorIdxs)
}

// Calculate reward at the end of each epoch.
func (s *Ethereum) PosvGetEpochReward(
	c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader, statedb *state.StateDB, logger log.Logger,
) (*posv.EpochReward, error) {
	epochRewards := &posv.EpochReward{}
	blockNumber := header.Number.Uint64()

	// Skip block 900 (1*epoch); first reward at block 1800 (2*epoch)
	if blockNumber <= posvConfig.Epoch {
		return epochRewards, nil
	}

	// Get initial reward
	initialRewardPerEpoch := (*big.Int)(vicConfig.RewardPerEpoch)
	totalReward := viction.CalcDefaultRewardPerBlock(initialRewardPerEpoch, blockNumber, posvConfig.BlocksPerYear())

	// Get additional reward for Saigon upgrade
	if config.IsSaigon(header.Number) && vicConfig.SaigonRewardPerEpoch != nil {
		saigonRewardPerEpoch := (*big.Int)(vicConfig.SaigonRewardPerEpoch)
		saigonReward := viction.CalcSaigonRewardPerBlock(saigonRewardPerEpoch, config.SaigonBlock, blockNumber, posvConfig.BlocksPerYear())
		totalReward = new(big.Int).Add(totalReward, saigonReward)
	}

	// Calculate rewards for validators and stakeholders
	validatorRewards, err := viction.CalcRewardsForValidators(c, config, posvConfig, vicConfig, header, totalReward, chain, logger)
	if err != nil {
		return nil, err
	}
	epochRewards.ValidatorRewards = validatorRewards

	// Use pre-transaction state for voter caps
	parentHeader := chain.GetHeader(header.ParentHash, blockNumber-1)
	var rewardState *state.StateDB
	if parentHeader != nil {
		rewardState, err = s.BlockChain().StateAt(parentHeader.Root)
		if err != nil {
			logger.Warn("PosvGetEpochReward: failed to get parent state, falling back to current state", "block", blockNumber, "err", err)
			rewardState = statedb
		}
	} else {
		rewardState = statedb
	}

	stakeholderRewards, nestedRewards, err := viction.CalcRewardsForStakeholders(c, config, posvConfig, vicConfig, header, validatorRewards, rewardState, logger)
	if err != nil {
		return nil, err
	}
	epochRewards.StakeholderRewards = stakeholderRewards
	epochRewards.Rewards = nestedRewards

	return epochRewards, nil
}

// Add balance rewards to the state (apply the rewards returned by PosvGetEpochReward).
func (s *Ethereum) PosvDistributeEpochRewards(
	header *types.Header, state *state.StateDB, epochReward *posv.EpochReward,
) error {
	blockNumber := header.Number.Uint64()

	if epochReward == nil {
		log.Debug("PosvAddBalanceRewards: no epoch rewards to apply", "block", blockNumber)
		return nil
	}
	if state == nil {
		return nil
	}

	// Apply stakeholder rewards to the state
	totalRewardDistributed := big.NewInt(0)
	rewardCount := 0

	for addr, amount := range epochReward.StakeholderRewards {
		if amount == nil || amount.Sign() <= 0 {
			continue
		}
		state.AddBalance(addr, amount)
		totalRewardDistributed.Add(totalRewardDistributed, amount)
		rewardCount++
	}

	log.Info("PosvAddBalanceRewards: applied epoch rewards", "block", blockNumber, "recipientCount", rewardCount, "totalReward", totalRewardDistributed.String())
	return nil
}

// Penalize validators for creating bad block or not creating block at all.
func (s *Ethereum) PosvGetPenalties(
	c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader, validators []common.Address,
) ([]common.Address, error) {
	if config.IsTIPSigning(header.Number) {
		return viction.PenalizeValidatorsTIPSigning(c, config, posvConfig, vicConfig, header, chain, validators)
	}
	return viction.PenalizeValidatorsDefault(s.BlockChain(), c, config, posvConfig, vicConfig, header, chain)
}

// Get eligble validators from the state.
func (s *Ethereum) PosvGetValidators(
	config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader,
) ([]common.Address, error) {
	if header == nil {
		return []common.Address{}, fmt.Errorf("header is nil")
	}
	// Guard against being called before eth.blockchain is assigned (e.g. during
	// NewBlockChain's internal VerifyHeader when SetBackend was called too early).
	bc := s.BlockChain()
	if bc == nil {
		return []common.Address{}, fmt.Errorf("blockchain not initialized (block %v)", header.Number)
	}

	state, err := bc.StateAt(header.Root)
	if err != nil {
		return []common.Address{}, fmt.Errorf("failed to get state at header root (block %v): %v", header.Number, err)
	}
	contracrAddress := vicConfig.ValidatorContract
	if contracrAddress == (common.Address{}) {
		return []common.Address{}, viction.ErrNoContractAddress
	}
	addresses := state.VicGetCandidates(contracrAddress)
	candidates := []*posv.ValidatorInfo{}
	for _, addr := range addresses {
		if addr == (common.Address{}) {
			continue
		}
		_, cap := state.VicGetValidatorInfo(contracrAddress, addr)
		candidates = append(candidates, &posv.ValidatorInfo{Address: addr, Capacity: cap})
	}

	if s.blockchain.Config().IsAtlas(header.Number) {
		sort.SliceStable(candidates, func(i, j int) bool {
			return candidates[i].Capacity.Cmp(candidates[j].Capacity) >= 0
		})
	} else {
		sortlgc.Slice(candidates, func(i, j int) bool {
			return candidates[i].Capacity.Cmp(candidates[j].Capacity) >= 0
		})
	}

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

// setupPosvBackend wires the POSV engine backend and the legacy TomoX/TomoZ engines
// for historical block replay (pre-Atlas sync).
// openTomoXDatabase opens the TomoX LevelDB.
//   - If --tomox.datadir is set, that explicit path is used directly.
//   - Otherwise the database is created at {datadir}/{instance}/tomox.
func openTomoXDatabase(cfg *Config, stack *node.Node) (ethdb.Database, error) {
	if cfg.TomoXDataDir != "" {
		log.Info("Opening TomoX database at custom path", "path", cfg.TomoXDataDir)
		return rawdb.NewLevelDBDatabase(cfg.TomoXDataDir, 256, 256, "eth/db/tomox/")
	}
	return stack.OpenDatabase("tomox", 256, 256, "eth/db/tomox/")
}

func (eth *Ethereum) setupPosvBackend(chainConfig *params.ChainConfig, stack *node.Node) error {
	if chainConfig.Posv == nil {
		return nil
	}

	posvEngine, ok := eth.engine.(*posv.Posv)
	if !ok {
		return fmt.Errorf("posv config present but engine is %T, expected *posv.Posv", eth.engine)
	}

	// Wire POSV backend
	posvEngine.SetBackend(eth)
	log.Info("PosvBackend set on Posv engine")
	if head := eth.blockchain.CurrentHeader(); head != nil {
		if err := eth.engine.VerifyHeader(eth.blockchain, head, true); err != nil {
			log.Warn("Head invalid after full POSV check", "number", head.Number, "hash", head.Hash(), "err", err,
				"hint", "often missing M2 attestor on post-epoch head; rewind or re-sync if chain stuck")
		}
	}

	// Initialize legacy TomoX/TomoZ engines for historical block replay.
	// Both share a single LevelDB ("tomox") since TradingStateDB and LendingStateDB
	// use independent trie roots and their key-spaces never collide.
	tomoxDb, err := openTomoXDatabase(eth.config, stack)
	if err != nil {
		log.Error("Failed to open TomoX database", "err", err)
		return nil
	}
	tomoxEngine := trading.NewWithDB(tomoxDb, eth.blockchain.Config())
	eth.blockchain.SetTradingEngine(tomoxEngine)

	lendingEngine := lending.New(tomoxDb, tomoxEngine, eth.blockchain.Config())
	eth.blockchain.SetLendingEngine(lendingEngine)

	return nil
}
