package eth

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/sortlgc"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/viction"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/legacy/lending"
	"github.com/ethereum/go-ethereum/legacy/trading"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

// Create a new block with attestor's signature. Only accept non-attested block.
func (s *Ethereum) PosvAttestBlock(
	block *types.Block,
) (*types.Block, error) {
	c := s.engine.(*posv.Posv)
	config := s.blockchain.Config()
	posvConfig := config.Posv

	header := block.Header()
	number := header.Number.Uint64()
	if number <= posvConfig.Epoch {
		return block, nil
	}
	// Only accept non-attested block.
	if len(header.Attestor) == posv.ExtraSeal {
		return nil, nil
	}
	eb, err := s.Etherbase()
	if err != nil || eb.IsZero() {
		return nil, nil
	}
	creator, err := c.Author(header)
	if err != nil {
		return nil, nil
	}
	checkpointHeader := posv.GetCheckpointHeader(posvConfig, header, s.blockchain, nil)
	valAttPairs, _, err := s.PosvGetCreatorAttestorPairs(config, posvConfig, config.Viction, header, checkpointHeader)
	if err != nil {
		return nil, nil
	}
	assigned, ok := valAttPairs[creator]
	if !ok || eb != assigned {
		return nil, nil
	}
	wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
	if wallet == nil || err != nil {
		return nil, nil
	}
	sig, err := wallet.SignData(accounts.Account{Address: eb}, accounts.MimetypePosv, posv.PosvRLP(header))
	if err != nil {
		return nil, err
	}
	attestedHeader := types.CopyHeader(header)
	attestedHeader.Attestor = make([]byte, len(sig))
	copy(attestedHeader.Attestor, sig)
	attestedBlock := block.WithSeal(attestedHeader)
	attestedBlock.ReceivedAt = block.ReceivedAt // preserve for propagation-latency metrics
	return attestedBlock, nil
}

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
	if state == nil || epochReward == nil {
		return nil
	}

	number := header.Number.Uint64()
	rewardAmount := big.NewInt(0)
	stakeholderCount := 0
	for addr, amount := range epochReward.StakeholderRewards {
		if amount == nil || amount.Sign() <= 0 {
			continue
		}
		state.AddBalance(addr, amount)
		rewardAmount.Add(rewardAmount, amount)
		stakeholderCount++
	}

	log.Info("[Backend] Distributed epoch rewards", "block", number, "stakeholderCount", stakeholderCount, "totalReward", rewardAmount.String())
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

// Create new Randomize transaction and submit to TxPool.
func (s *Ethereum) PosvRandomNumber(
	block *types.Block,
) error {
	c := s.engine.(*posv.Posv)
	config := s.blockchain.Config()
	vicConfig := config.Viction
	posvConfig := config.Posv
	number := block.NumberU64()

	if !config.IsTIPRandomize(block.Number()) {
		log.Info("[Backend][RandomNumber] Randomize not enabled", "number", block.NumberU64())
		return nil
	}
	blockOfEpoch := number % posvConfig.Epoch
	commitPhase := blockOfEpoch > 0 && blockOfEpoch >= vicConfig.RandomizerCommitNthBlock && blockOfEpoch < vicConfig.RandomizerRevealNthBlock
	if commitPhase {
		chaindb := s.ChainDb()
		exists, _ := chaindb.Has(viction.RandomizeKeyName)
		if exists {
			// Already committed this epoch.
			return nil
		}

		eb, err := s.Etherbase()
		// Only validator can sign block.
		if err != nil || eb.IsZero() {
			return nil
		}
		snap, err := c.GetSnapshot(s.blockchain, block.Header())
		if err != nil {
			return nil
		}
		if _, ok := snap.Signers[eb]; !ok {
			log.Debug("[Backend][RandomNumber] Not a validator in this epoch", "etherbase", eb, "number", number)
			return nil
		}
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Warn("[Backend][RandomNumber] Etherbase key is not available", "etherbase", eb, "err", err)
			return nil
		}

		statedb, err := s.blockchain.State()
		if err != nil {
			return err
		}
		nonce := statedb.GetNonce(eb)
		pendingTxs, _ := s.txPool.Pending()
		nonce += uint64(len(pendingTxs[eb]))

		key := viction.GenerateRandomKey()
		secret, err := viction.GenerateRandomNumber(posvConfig.Epoch, key)
		if err != nil {
			log.Error("[Backend][RandomNumber] Failed to generate RandomNumber", "number", number, "err", err)
			return err
		}
		tx := viction.CreateSetRandomizeSecretTransaction(nonce, vicConfig.RandomizerContract, secret)
		signedTx, err := wallet.SignTx(accounts.Account{Address: eb}, tx, config.ChainID)
		if err != nil {
			return err
		}
		if err := s.txPool.AddLocal(signedTx); err != nil {
			if err == core.ErrReplaceUnderpriced || err == core.ErrAlreadyKnown {
				log.Info("[Backend][RandomNumber] RandomizeSecret transaction is duplicated", number, "blockHash", block.Hash(), "etherbase", eb, "txHash", signedTx.Hash(), "nonce", nonce)
				return nil
			}
			log.Warn("[Backend][RandomNumber] Failed to submit RandomizeSecret transaction", number, "blockHash", block.Hash(), "etherbase", eb, "txHash", signedTx.Hash(), "nonce", nonce, "err", err)
			return err
		}
		if err := chaindb.Put(viction.RandomizeKeyName, key); err != nil {
			log.Error("[Backend][RandomNumber] Failed to store RandomizeKey to database", "number", number, "err", err)
			return err
		}
		log.Info("[Backend][RandomNumber] Submitted RandomizeSecret transaction", "number", number, "blockHash", block.Hash(), "etherbase", eb, "txHash", signedTx.Hash(), "nonce", nonce)
		return nil
	}
	openingPhase := blockOfEpoch > 0 && blockOfEpoch >= vicConfig.RandomizerRevealNthBlock && blockOfEpoch <= vicConfig.RandomizerFinaleNthBlock
	if openingPhase {
		chaindb := s.ChainDb()
		key, err := chaindb.Get(viction.RandomizeKeyName)
		if err != nil || len(key) == 0 {
			// Either already revealed or never committed in this epoch.
			return nil
		}

		eb, err := s.Etherbase()
		// Only validator can sign block.
		if err != nil || eb.IsZero() {
			return nil
		}
		snap, err := c.GetSnapshot(s.blockchain, block.Header())
		if err != nil {
			return nil
		}
		if _, ok := snap.Signers[eb]; !ok {
			log.Debug("[Backend][RandomNumber] Not a validator in this epoch", "etherbase", eb, "number", number)
			return nil
		}
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Warn("[Backend][RandomNumber] Etherbase key is not available", "etherbase", eb, "err", err)
			return nil
		}

		statedb, err := s.blockchain.State()
		if err != nil {
			return err
		}
		nonce := statedb.GetNonce(eb)
		pendingTxs, _ := s.txPool.Pending()
		nonce += uint64(len(pendingTxs[eb]))

		tx := viction.CreateSetRandomizeOpeningTransaction(nonce, vicConfig.RandomizerContract, key)
		signedTx, err := wallet.SignTx(accounts.Account{Address: eb}, tx, config.ChainID)
		if err != nil {
			return err
		}
		if err := s.txPool.AddLocal(signedTx); err != nil {
			if err == core.ErrReplaceUnderpriced || err == core.ErrAlreadyKnown {
				log.Info("[Backend][RandomNumber] RandomizeOpening transaction is duplicated", number, "blockHash", block.Hash(), "etherbase", eb, "txHash", signedTx.Hash(), "nonce", nonce)
				return nil
			}
			log.Warn("[Backend][RandomNumber] Failed to submit RandomizeOpening transaction", number, "blockHash", block.Hash(), "etherbase", eb, "txHash", signedTx.Hash(), "nonce", nonce, "err", err)
			return err
		}
		if err := chaindb.Delete(viction.RandomizeKeyName); err != nil {
			log.Error("[Backend][RandomNumber] Failed to delete RandomizeKey in database", "number", number, "err", err)
		}
		log.Info("[Backend][RandomNumber] Submitted RandomizeSecret transaction", "number", number, "blockHash", block.Hash(), "etherbase", eb, "txHash", signedTx.Hash(), "nonce", nonce)
	}
	return nil
}

// Create new BlockSign transaction and submit to TxPool.
func (s *Ethereum) PosvSignBlock(
	block *types.Block,
) error {
	c := s.engine.(*posv.Posv)
	config := s.blockchain.Config()
	vicConfig := config.Viction
	number := block.NumberU64()

	// Pre-TIP2019: Emit SignBlock transaction every block.
	// TIP2019: Emit SignBlock transaction every *vicConfig.ValidatorSignInterval* blocks.
	if config.IsTIP2019(block.Number()) && number%vicConfig.ValidatorSignInterval != 0 {
		log.Info("[Backend][SignBlock] Skipped sign block: not interval block", "number", block.NumberU64(), "inveral", vicConfig.ValidatorSignInterval, "blockOfInterval", number%vicConfig.ValidatorSignInterval)
		return nil
	}

	eb, err := s.Etherbase()
	// Only validator can sign block.
	if err != nil || eb.IsZero() {
		return nil
	}
	snap, err := c.GetSnapshot(s.blockchain, block.Header())
	if err != nil {
		return nil
	}
	if _, ok := snap.Signers[eb]; !ok {
		log.Debug("[Backend][SignBlock] Not a validator in this epoch", "etherbase", eb, "number", block.NumberU64())
		return nil
	}
	wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
	if wallet == nil || err != nil {
		log.Warn("[Backend][SignBlock] Etherbase key is not available", "etherbase", eb, "err", err)
		return nil
	}

	statedb, err := s.blockchain.State()
	if err != nil {
		return err
	}
	nonce := statedb.GetNonce(eb)
	tx := viction.CreateBlockSignTransaction(nonce, config.Viction.ValidatorBlockSignContract, block.Number(), block.Hash())
	signedTx, err := wallet.SignTx(accounts.Account{Address: eb}, tx, config.ChainID)
	if err != nil {
		return err
	}
	if err := s.txPool.AddLocal(signedTx); err != nil {
		if err == core.ErrReplaceUnderpriced || err == core.ErrAlreadyKnown {
			log.Info("[Backend][SignBlock] BlockSign transaction is duplicated", number, "blockHash", block.Hash(), "etherbase", eb, "txHash", signedTx.Hash(), "nonce", nonce)
			return nil
		}
		log.Warn("[Backend][SignBlock] Failed to submit BlockSign transaction", number, "blockHash", block.Hash(), "etherbase", eb, "txHash", signedTx.Hash(), "nonce", nonce, "err", err)
		return err
	}
	log.Info("[Backend][SignBlock] Submitted BlockSign transaction", "number", number, "blockHash", block.Hash(), "etherbase", eb, "txHash", signedTx.Hash(), "nonce", nonce)
	return nil
}

// Attach services required by Viction blockchain.
func (s *Ethereum) setupPosvBackend(chainConfig *params.ChainConfig, stack *node.Node) error {
	if !chainConfig.IsPosv() {
		return nil
	}

	posvEngine, ok := s.engine.(*posv.Posv)
	if !ok {
		return fmt.Errorf("posv config present but engine is %T, expected *posv.Posv", s.engine)
	}

	posvEngine.SetBackend(s)
	log.Info("[Backend] Set current backend reference to PoSV engine.")
	s.protocolManager.blockFetcher.SetPosvBackend(s)
	log.Info("[Backend] Set current backend reference to BlockFetcher.")

	tradingStatedb, err := openTradingDatabase(stack)
	if err != nil {
		log.Error("failed to open Trading database", "err", err)
		return nil
	}
	tradingEngine := trading.NewWithDB(tradingStatedb, s.blockchain.Config())
	s.blockchain.SetTradingEngine(tradingEngine)

	lendingEngine := lending.New(tradingStatedb, tradingEngine, s.blockchain.Config())
	s.blockchain.SetLendingEngine(lendingEngine)

	return nil
}

// Open Trading LevelDB for storing trasnsient data required for native trading/lending engine
func openTradingDatabase(stack *node.Node) (ethdb.Database, error) {
	return stack.OpenDatabase("trading", 256, 256, "eth/db/trading/")
}
