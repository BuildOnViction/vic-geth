package eth

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/viction"
	"github.com/ethereum/go-ethereum/legacy/tomox"
	"github.com/ethereum/go-ethereum/legacy/tomoxlending"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/sortlgc"
)

const SignMethodHex = "e341eaa4"

// Get attestors from list of validators at checkpoint block.
func (s *Ethereum) PosvGetAttestors(vicConfig *params.VictionConfig, header *types.Header, validators []common.Address,
) ([]int64, error) {
	state, err := s.BlockChain().State()
	if err != nil {
		return nil, err
	}
	return viction.GetAttestors(vicConfig, validators, state)
}

// Get block signers from the state.
func (s *Ethereum) PosvGetBlockSignData(config *params.ChainConfig, vicConfig *params.VictionConfig, header *types.Header,
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
func (s *Ethereum) PosvGetCreatorAttestorPairs(c *posv.Posv, config *params.ChainConfig,
	header, checkpointHeader *types.Header,
) (map[common.Address]common.Address, uint64, error) {
	pairs, offset, err := viction.GetCreatorAttestorPairs(c, config, config.Posv, header, checkpointHeader)
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

// PosvGetEpochReward calculates and distributes reward at checkpoint block.
func (s *Ethereum) PosvGetEpochReward(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
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

	stakeholderRewards, err := viction.CalcRewardsForStakeholders(c, config, posvConfig, vicConfig, header, validatorRewards, rewardState, logger)
	if err != nil {
		return nil, err
	}
	epochRewards.StakholderRewards = stakeholderRewards

	return epochRewards, nil
}

// PosvAddBalanceRewards applies epoch rewards to the state by adding balances to all stakeholders.
// It does NOT recalculate; caller should pass the epochReward returned by PosvGetEpochReward.
func (s *Ethereum) PosvDistributeEpochRewards(header *types.Header, state *state.StateDB, epochReward *posv.EpochReward) error {
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

	for addr, amount := range epochReward.StakholderRewards {
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

// Get list of validators creating bad block or not creating block at all.
func (s *Ethereum) PosvGetPenalties(c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig,
	header *types.Header,
	chain consensus.ChainReader,
	validators []common.Address,
) ([]common.Address, error) {
	if config.IsTIPSigning(header.Number) {
		return viction.PenalizeValidatorsTIPSigning(c, config, posvConfig, vicConfig, header, chain, validators)
	}
	return viction.PenalizeValidatorsDefault(s.BlockChain(), c, config, posvConfig, vicConfig, header, chain)
}

// Get eligble validators from the state.
func (s *Ethereum) PosvGetValidators(vicConfig *params.VictionConfig, header *types.Header, chain consensus.ChainReader,
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

	// Initialize legacy TomoX trading engine for historical block replay.
	// Required to sync pre-Atlas blocks containing TomoX order matching transactions (0x91).
	tradingDb, err := stack.OpenDatabase("tomox", 256, 256, "eth/db/tomox/")
	if err != nil {
		log.Error("Failed to open TomoX trading database", "err", err)
		return nil
	}
	tomoxEngine := tomox.NewWithDB(tradingDb, eth.blockchain.Config())
	eth.blockchain.SetTradingEngine(tomoxEngine)

	// Initialize legacy TomoZ lending engine for historical block replay.
	// Required to sync pre-Atlas blocks containing TomoZ lending order transactions (0x93).
	// Must share the same tomoxEngine instance so lending price lookups hit the same trading state.
	lendingDb, err := stack.OpenDatabase("tomoxlending", 256, 256, "eth/db/tomoxlending/")
	if err != nil {
		log.Error("Failed to open TomoZ lending database", "err", err)
		return nil
	}
	lendingEngine := tomoxlending.New(lendingDb, tomoxEngine, eth.blockchain.Config())
	eth.blockchain.SetLendingEngine(lendingEngine)

	return nil
}
