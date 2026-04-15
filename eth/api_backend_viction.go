// Copyright 2026 The Vic-geth Authors
package eth

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/eth/viction"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

func (s *EthAPIBackend) GetRewardByHash(hash common.Hash) *posv.EpochReward {
	header := s.eth.blockchain.GetHeaderByHash(hash)
	if header == nil || header.Number.Uint64()%s.eth.blockchain.Config().Posv.Epoch != 0 {
		return nil
	}
	engine := s.Engine().(*posv.Posv)
	statedb, err := s.eth.blockchain.StateAt(header.Root)
	if err != nil {
		log.Info("Failed to get state at", "hash", hash, "error", err)
		return nil
	}
	epochReward, err := s.eth.PosvGetEpochReward(
		engine,
		s.eth.blockchain.Config(),
		s.eth.blockchain.Config().Posv,
		s.eth.blockchain.Config().Viction,
		header,
		s.eth.blockchain,
		statedb,
		log.New(),
	)
	if err != nil {
		return nil
	}
	if epochReward == nil {
		return nil
	}
	return epochReward

}

// GetVotersRewards return a map of voters of snapshot at given block hash
// there is a function engine.HookReward nearly does the same thing but
// it does change the stateDB too - so can't use it here
// Steps:
// 1. Checking back to state of last checkpoint
// 2. Get list signers + reward at that checkpoint
// 3. Find out the list signers_reward for input masternode's reward
// 4. Calculate voters's rewards for input masternode
func (b *EthAPIBackend) GetVotersRewards(masternodeAddr common.Address, blockHash common.Hash) map[common.Address]*big.Int {
	chain := b.eth.blockchain
	block := chain.GetBlockByHash(blockHash)
	if block == nil {
		return nil
	}
	number := block.Number().Uint64()
	engine := b.Engine().(*posv.Posv)
	foundationWalletAddr := b.ChainConfig().Viction.RewardFoundationAddress
	lastCheckpointNumber := number - (number % b.ChainConfig().Posv.Epoch) - b.ChainConfig().Posv.Epoch // calculate for 2 epochs ago
	lastCheckpointBlock := chain.GetBlockByNumber(lastCheckpointNumber)
	rCheckpoint := b.ChainConfig().Posv.Epoch

	state, err := chain.StateAt(lastCheckpointBlock.Root())
	if err != nil {
		fmt.Println("ERROR Trying to getting state at", lastCheckpointNumber, " Error ", err)
		return nil
	}

	if foundationWalletAddr == (common.Address{}) {
		log.Error("Foundation Wallet Address is empty", "error", foundationWalletAddr)
		return nil
	}

	if lastCheckpointNumber <= 0 || lastCheckpointNumber-rCheckpoint <= 0 || foundationWalletAddr == (common.Address{}) {
		return nil
	}

	// Get initial reward
	rewardPerEpoch := (*big.Int)(b.ChainConfig().Viction.RewardPerEpoch)
	initialRewardPerEpoch := new(big.Int).Mul(rewardPerEpoch, new(big.Int).SetUint64(params.Ether))
	chainReward := viction.CalcDefaultRewardPerBlock(initialRewardPerEpoch, lastCheckpointNumber, b.ChainConfig().Posv.BlocksPerYear())

	// Get additional reward for Saigon upgrade
	if chain.Config().IsSaigon(block.Number()) {
		saigonRewardPerEpoch := new(big.Int).Mul((*big.Int)(b.ChainConfig().Viction.SaigonRewardPerEpoch), new(big.Int).SetUint64(params.Ether))
		chainReward = new(big.Int).Add(chainReward, viction.CalcSaigonRewardPerBlock(saigonRewardPerEpoch, b.ChainConfig().SaigonBlock, lastCheckpointNumber, b.ChainConfig().Posv.BlocksPerYear()))
	}

	if err != nil {
		log.Crit("Fail to get signers for reward checkpoint", "error", err)
		return nil
	}

	rewardForValidators, err := viction.CalcRewardsForValidators(engine, b.ChainConfig(), b.ChainConfig().Posv, b.ChainConfig().Viction, lastCheckpointBlock.Header(), chainReward, chain, log.New())
	if err != nil {
		log.Crit("Fail to calculate reward for signers", "error", err)
		return nil
	}

	if len(rewardForValidators) <= 0 {
		return nil
	}

	// Add reward for coin voters of input validator.
	voterResults, err := viction.CalcRewardsForStakeholders(engine, b.ChainConfig(), b.ChainConfig().Posv, b.ChainConfig().Viction, lastCheckpointBlock.Header(), rewardForValidators, state, log.New())
	if err != nil {
		log.Crit("Fail to calculate reward for stakeholders", "error", err)
		return nil
	}

	return voterResults
}

// GetVotersCap return all voters's capability at a checkpoint
func (b *EthAPIBackend) GetVotersCap(checkpoint *big.Int, masterAddr common.Address, voters []common.Address) map[common.Address]*big.Int {
	chain := b.eth.blockchain
	checkpointBlock := chain.GetBlockByNumber(checkpoint.Uint64())
	state, err := chain.StateAt(checkpointBlock.Root())

	if err != nil {
		fmt.Println("ERROR Trying to getting state at", checkpoint, " Error ", err)
		return nil
	}

	voterCaps := make(map[common.Address]*big.Int)
	for _, voteAddr := range voters {
		voterCap := state.VicGetValidatorVoterCap(b.ChainConfig().Viction.ValidatorContract, masterAddr, voteAddr)
		voterCaps[voteAddr] = voterCap
	}
	return voterCaps
}

// GetMasternodesCap return a cap of all masternode at a checkpoint
func (b *EthAPIBackend) GetMasternodesCap(checkpoint uint64) map[common.Address]*big.Int {
	checkpointBlock := b.eth.blockchain.GetBlockByNumber(checkpoint)
	state, err := b.eth.blockchain.StateAt(checkpointBlock.Root())

	if err != nil {
		fmt.Println("ERROR Trying to getting state at", checkpoint, " Error ", err)
		return nil
	}

	validators := state.VicGetCandidates(b.ChainConfig().Viction.ValidatorContract)

	validatorCaps := make(map[common.Address]*big.Int)
	for _, validator := range validators {
		_, validatorCap := state.VicGetValidatorInfo(b.ChainConfig().Viction.ValidatorContract, validator)
		validatorCaps[validator] = validatorCap
	}

	return validatorCaps
}

func (b *EthAPIBackend) AreTwoBlockSamePath(bh1 common.Hash, bh2 common.Hash) bool {
	return b.eth.blockchain.AreTwoBlockSamePath(bh1, bh2)
}

// GetBlocksHashCache returns cached block hash candidates at a given height.
func (b *EthAPIBackend) GetBlocksHashCache(blockNr uint64) []common.Hash {
	return b.eth.blockchain.GetBlocksHashFromBlockCache(blockNr)
}

// GetEpochDuration return latest generating velocity epoch by minute
// ie 30min for each epoch
func (b *EthAPIBackend) GetEpochDuration() *big.Int {
	chain := b.eth.blockchain
	block := chain.CurrentBlock()
	number := block.Number().Uint64()
	lastCheckpointNumber := number - (number % b.ChainConfig().Posv.Epoch)
	lastCheckpointBlockTime := chain.GetBlockByNumber(lastCheckpointNumber).Time()
	secondToLastCheckpointNumber := lastCheckpointNumber - b.ChainConfig().Posv.Epoch
	secondToLastCheckpointBlockTime := chain.GetBlockByNumber(secondToLastCheckpointNumber).Time()

	return new(big.Int).Sub(new(big.Int).SetUint64(lastCheckpointBlockTime), new(big.Int).SetUint64(secondToLastCheckpointBlockTime))
}
