package eth

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/victionapi"
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
	eb, wallet, err := s.getEtherbaseWallet()
	if err != nil || eb.IsZero() {
		return nil, nil
	}
	creator, err := c.Author(header)
	if err != nil {
		return nil, nil
	}
	checkpointHeader := posv.GetCheckpointHeader(posvConfig, header, nil, s.blockchain)
	valAttPairs, _, err := s.PosvGetCreatorAttestorPairs(config, posvConfig, config.Viction, header, checkpointHeader)
	if err != nil {
		return nil, nil
	}
	assigned, ok := valAttPairs[creator]
	if !ok || eb != assigned {
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
	return victionapi.GetAttestorsFromState(vicConfig, validators, state)
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

	// Only successful signing txs count toward rewards and penalties.
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
	if config.Viction == nil || checkpointHeader == nil {
		return nil, 0, victionapi.ErrInvalidAttestorList
	}
	pairs, offset, err := victionapi.GetCreatorAttestorPairsFromCheckpointHeader(config, config.Posv, header, checkpointHeader)
	if err == nil || err != victionapi.ErrNoValidator {
		return pairs, offset, err
	}

	number := header.Number.Uint64()
	checkpointNumber := checkpointHeader.Number.Uint64()
	log.Warn("[Backend][GetCreatorAttestorPairs] Get Creator-Attestor Pairs from checkpoint header failed. Retry with state", "checkpoint", checkpointNumber, "number", number)
	stateAtCheckpoint, err2 := s.blockchain.StateAt(checkpointHeader.Root)
	if err2 != nil {
		log.Warn("[Backend][GetCreatorAttestorPairs]: failed to get state at checkpoint", "checkpoint", checkpointNumber, "err", err2)
		return nil, 0, err
	}
	return victionapi.GetCreatorAttestorPairsFromState(config, posvConfig, header, checkpointHeader, stateAtCheckpoint)
}

// Calculate rewards at the end of each epoch.
func (s *Ethereum) PosvGetEpochRewards(
	c *posv.Posv, config *params.ChainConfig, posvConfig *params.PosvConfig, vicConfig *params.VictionConfig, header *types.Header,
	chain consensus.ChainReader, statedb *state.StateDB, logger log.Logger,
) (*posv.EpochReward, error) {
	epochRewards := &posv.EpochReward{}
	number := header.Number.Uint64()

	// First epoch won't include any rewards
	if !posvConfig.IsCheckpointBlock(number) || number <= posvConfig.Epoch {
		return epochRewards, nil
	}

	// Get initial reward
	initialRewardPerEpoch := (*big.Int)(vicConfig.RewardPerEpoch)
	totalReward := victionapi.CalcDefaultRewardPerBlock(initialRewardPerEpoch, number, posvConfig.BlocksPerYear())

	// Get additional reward for Saigon upgrade
	if config.IsSaigon(header.Number) && vicConfig.SaigonRewardPerEpoch != nil {
		saigonRewardPerEpoch := (*big.Int)(vicConfig.SaigonRewardPerEpoch)
		saigonReward := victionapi.CalcSaigonRewardPerBlock(saigonRewardPerEpoch, config.SaigonBlock, number, posvConfig.BlocksPerYear())
		totalReward = new(big.Int).Add(totalReward, saigonReward)
	}

	// Calculate rewards for validators and stakeholders
	validatorRewards, err := victionapi.CalcRewardsForValidators(c, config, posvConfig, vicConfig, header, totalReward, chain, logger)
	if err != nil {
		return nil, err
	}
	epochRewards.ValidatorRewards = validatorRewards

	// Use parent state for voter caps
	preCheckpointHeader := chain.GetHeader(header.ParentHash, number-1)
	preCheckpointState, err := s.BlockChain().StateAt(preCheckpointHeader.Root)
	if err != nil {
		logger.Warn("[Backend][GetEpochReward]: failed to get preCheckpoint state", "number", number-1, "hash", header.ParentHash, "err", err)
		return epochRewards, err
	}

	stakeholderRewards, nestedRewards, err := victionapi.CalcRewardsForStakeholders(c, config, posvConfig, vicConfig, header, validatorRewards, preCheckpointState, logger)
	if err != nil {
		return nil, err
	}
	epochRewards.StakeholderRewards = stakeholderRewards
	epochRewards.Rewards = nestedRewards

	return epochRewards, nil
}

// Add balance rewards to the state (apply the rewards returned by PosvGetEpochReward).
func (s *Ethereum) PosvDistributeEpochRewards(
	header *types.Header, epochReward *posv.EpochReward, state *state.StateDB,
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
		return victionapi.PenalizeValidatorsTIPSigning(c, config, posvConfig, vicConfig, header, chain, validators)
	}
	return victionapi.PenalizeValidatorsDefault(s.BlockChain(), c, config, posvConfig, vicConfig, header, chain)
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

	statedb, err := bc.StateAt(header.Root)
	if err != nil {
		return []common.Address{}, fmt.Errorf("failed to get state at header root (block %v): %v", header.Number, err)
	}
	return victionapi.GetValidatorsFromState(vicConfig, s.blockchain.Config().IsAtlas(header.Number), statedb)
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
	commitPhase := vicConfig.IsRandomizerCommitPhase(blockOfEpoch)
	if commitPhase {
		chaindb := s.ChainDb()
		exists, _ := chaindb.Has(victionapi.RandomizeKeyName)
		if exists {
			// Already committed this epoch.
			return nil
		}

		eb, wallet, err := s.getEtherbaseWallet()
		if err != nil || eb.IsZero() {
			return nil
		}
		if !s.isEligibleValidator(c, eb, block.Header()) {
			log.Debug("[Backend][RandomNumber] Not a validator in this epoch", "etherbase", eb, "number", number)
			return nil
		}

		nonce := s.nextNonce(eb)

		key := victionapi.GenerateRandomKey()
		secret, err := victionapi.GenerateRandomNumber(posvConfig.Epoch, key)
		if err != nil {
			log.Error("[Backend][RandomNumber] Failed to generate RandomNumber", "number", number, "err", err)
			return err
		}
		tx := victionapi.CreateSetRandomizeSecretTransaction(nonce, vicConfig.RandomizerContract, secret)
		if err := s.signAndSubmitTransaction(eb, wallet, tx, config.ChainID, "RandomNumber"); err != nil {
			return err
		}
		if err := chaindb.Put(victionapi.RandomizeKeyName, key); err != nil {
			log.Error("[Backend][RandomNumber] Failed to store RandomizeKey to database", "number", number, "err", err)
			return err
		}
		return nil
	}
	openingPhase := vicConfig.IsRandomizerOpeningPhase(blockOfEpoch)
	if openingPhase {
		chaindb := s.ChainDb()
		key, err := chaindb.Get(victionapi.RandomizeKeyName)
		if err != nil || len(key) == 0 {
			// Either already revealed or never committed in this epoch.
			return nil
		}

		eb, wallet, err := s.getEtherbaseWallet()
		if err != nil || eb.IsZero() {
			return nil
		}
		if !s.isEligibleValidator(c, eb, block.Header()) {
			log.Debug("[Backend][RandomNumber] Not a validator in this epoch", "etherbase", eb, "number", number)
			return nil
		}

		nonce := s.nextNonce(eb)

		tx := victionapi.CreateSetRandomizeOpeningTransaction(nonce, vicConfig.RandomizerContract, key)
		if err := s.signAndSubmitTransaction(eb, wallet, tx, config.ChainID, "RandomNumber"); err != nil {
			return err
		}
		if err := chaindb.Delete(victionapi.RandomizeKeyName); err != nil {
			log.Error("[Backend][RandomNumber] Failed to delete RandomizeKey in database", "number", number, "err", err)
		}
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

	eb, wallet, err := s.getEtherbaseWallet()
	if err != nil || eb.IsZero() {
		return nil
	}
	if !s.isEligibleValidator(c, eb, block.Header()) {
		log.Debug("[Backend][SignBlock] Not a validator in this epoch", "etherbase", eb, "number", block.NumberU64())
		return nil
	}

	nonce := s.nextNonce(eb)
	pendingTxs, _ := s.txPool.Pending()

	expectedData := victionapi.CreateBlockSignData(block.Number(), block.Hash())
	contractAddr := config.Viction.ValidatorBlockSignContract
	for _, tx := range pendingTxs[eb] {
		if tx.To() != nil && *tx.To() == contractAddr && bytes.Equal(tx.Data(), expectedData) {
			log.Info("[Backend][SignBlock] Skipped. BlockSign transaction already exists in pool", number, "blockHash", block.Hash(), "etherbase", eb, "nonce", tx.Nonce())
			return nil
		}
	}

	tx := victionapi.CreateBlockSignTransaction(nonce, config.Viction.ValidatorBlockSignContract, block.Number(), block.Hash())
	if err := s.signAndSubmitTransaction(eb, wallet, tx, config.ChainID, "SignBlock"); err != nil {
		return err
	}
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

// Get current etherbase altogether with it signing wallet.
func (s *Ethereum) getEtherbaseWallet() (common.Address, accounts.Wallet, error) {
	eb, err := s.Etherbase()
	if err != nil || eb.IsZero() {
		return common.Address{}, nil, err
	}
	wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
	if wallet == nil || err != nil {
		return common.Address{}, nil, err
	}
	return eb, wallet, nil
}

// Check if given etherbase is eligble signer for given header.
func (s *Ethereum) isEligibleValidator(c *posv.Posv, eb common.Address, header *types.Header) bool {
	snap, err := c.GetSnapshot(s.blockchain, header)
	if err != nil {
		return false
	}
	_, ok := snap.Signers[eb]
	return ok
}

// Return available nonce for the given address, with txPool awareness.
func (s *Ethereum) nextNonce(addr common.Address) uint64 {
	statedb, err := s.blockchain.State()
	if err != nil {
		return 0
	}
	nonce := statedb.GetNonce(addr)
	pendingTxs, _ := s.txPool.Pending()
	nonce += uint64(len(pendingTxs[addr]))
	return nonce
}

// Sign and submit given transaction to local txPool.
func (s *Ethereum) signAndSubmitTransaction(eb common.Address, wallet accounts.Wallet, tx *types.Transaction, chainID *big.Int, txLabel string) error {
	signedTx, err := wallet.SignTx(accounts.Account{Address: eb}, tx, chainID)
	if err != nil {
		return err
	}
	if err := s.txPool.AddLocal(signedTx); err != nil {
		if err == core.ErrReplaceUnderpriced || err == core.ErrAlreadyKnown {
			log.Info(fmt.Sprintf("[Backend][%s] Transaction is duplicated", txLabel),
				"txHash", signedTx.Hash(), "etherbase", eb, "nonce", signedTx.Nonce())
			return nil
		}
		log.Warn(fmt.Sprintf("[Backend][%s] Failed to submit transaction", txLabel),
			"txHash", signedTx.Hash(), "etherbase", eb, "nonce", signedTx.Nonce(), "err", err)
		return err
	}
	log.Info(fmt.Sprintf("[Backend][%s] Submitted transaction", txLabel),
		"txHash", signedTx.Hash(), "etherbase", eb, "nonce", signedTx.Nonce())
	return nil
}

// Open Trading LevelDB for storing trasnsient data required for native trading/lending engine
func openTradingDatabase(stack *node.Node) (ethdb.Database, error) {
	return stack.OpenDatabase("trading", 256, 256, "eth/db/trading/")
}
