package posv

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	extraVanity = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal   = 65 // Fixed number of extra-data suffix bytes reserved for signer seal
)

// rewardLog tracks signatures and reward for a signer during a checkpoint period.
type rewardLog struct {
	Sign   uint64
	Reward *big.Int
}

// rewardForCheckpoint distributes block rewards at reward checkpoint blocks.
// It is called from Finalize when block.Number % RewardCheckpoint == 0.
func (c *Posv) rewardForCheckpoint(chain consensus.ChainHeaderReader, statedb *state.StateDB, header *types.Header) error {
	number := header.Number.Uint64()
	chainConfig := chain.Config()
	rCheckpoint := chainConfig.Posv.Epoch
	foundationWalletAddr := chainConfig.Viction.RewardFoundationAddress

	if foundationWalletAddr == (common.Address{}) {
		log.Error("Foundation Wallet Address is empty")
		return nil // Don't fail block processing, just skip rewards
	}

	// Skip block 900 (1*epoch); first reward at block 1800 (2*epoch)
	if number <= rCheckpoint {
		return nil
	}

	// Calculate initial reward per epoch
	initialRewardPerEpoch := (*big.Int)(chainConfig.Viction.RewardPerEpoch)
	blocksPerYear := chainConfig.Posv.BlocksPerYear()
	chainReward := calcInitialReward(initialRewardPerEpoch, number, blocksPerYear)

	// Add Saigon additional reward if applicable
	if chainConfig.IsSaigon(header.Number) && chainConfig.Viction != nil && chainConfig.Viction.SaigonRewardPerEpoch != nil {
		saigonRewardPerEpoch := new(big.Int).Mul(
			(*big.Int)(chainConfig.Viction.SaigonRewardPerEpoch),
			new(big.Int).SetUint64(params.Ether),
		)
		chainReward = new(big.Int).Add(
			chainReward,
			calcSaigonReward(saigonRewardPerEpoch, chainConfig.SaigonBlock, number, blocksPerYear),
		)
	}

	// Get signers for this checkpoint period
	totalSigner := new(uint64)
	signers, err := c.getRewardForCheckpoint(chain, header, rCheckpoint, totalSigner)
	if err != nil {
		log.Error("Failed to get signers for reward checkpoint", "error", err)
		return nil // Don't fail block processing
	}

	// Calculate rewards per signer
	rewardSigners := calculateRewardForSigner(chainReward, signers, *totalSigner)

	// Distribute rewards to masternode owners, voters, and foundation
	// Note: In victionchain, parentState is used for voter reads.
	// In vic-geth, we use statedb directly since voter data is stable within a block.
	if len(signers) > 0 {
		for signer, calcReward := range rewardSigners {
			rewards := getRewardBalancesRate(
				foundationWalletAddr, statedb, signer, calcReward, number,
				chainConfig,
			)
			for holder, reward := range rewards {
				statedb.AddBalance(holder, reward)
			}
		}
	}

	log.Info("Distributed rewards at checkpoint", "block", number, "reward", chainReward, "signers", len(signers))
	return nil
}

// getRewardForCheckpoint scans the epoch to count block signers.
func (c *Posv) getRewardForCheckpoint(chain consensus.ChainHeaderReader, header *types.Header, rCheckpoint uint64, totalSigner *uint64) (map[common.Address]*rewardLog, error) {
	number := header.Number.Uint64()
	prevCheckpoint := number - (rCheckpoint * 2)
	startBlockNumber := prevCheckpoint + 1
	endBlockNumber := startBlockNumber + rCheckpoint - 1
	signers := make(map[common.Address]*rewardLog)
	mapBlkHash := map[uint64]common.Hash{}

	// Try to get ChainReader for block access (needed for tx caching)
	chainReader, hasBlocks := chain.(consensus.ChainReader)

	// Collect signer data from block-signer transactions
	data := make(map[common.Hash][]common.Address)
	h := header
	for i := prevCheckpoint + (rCheckpoint * 2) - 1; i >= startBlockNumber; i-- {
		h = chain.GetHeaderByHash(h.ParentHash)
		if h == nil {
			break
		}
		mapBlkHash[i] = h.Hash()

		signData, ok := c.BlockSigners.Get(h.Hash())
		if !ok && hasBlocks {
			// Cache miss - get block and cache signer txs
			block := chainReader.GetBlock(h.Hash(), i)
			if block != nil {
				blockSignAddr := common.Address{}
				if cfg := chain.Config(); cfg != nil && cfg.Viction != nil {
					blockSignAddr = cfg.Viction.ValidatorBlockSignContract
				}
				signData = c.CacheSigner(h.Hash(), block.Transactions(), blockSignAddr)
			} else {
				continue
			}
		} else if !ok {
			continue
		}

		txs, ok := signData.([]*types.Transaction)
		if !ok {
			continue
		}
		for _, tx := range txs {
			txData := tx.Data()
			if len(txData) < 68 {
				continue
			}
			blkHash := common.BytesToHash(txData[len(txData)-32:])

			signer := types.MakeSigner(chain.Config(), h.Number)
			msg, err := tx.AsMessage(signer)
			if err != nil {
				continue
			}
			from := msg.From()
			data[blkHash] = append(data[blkHash], from)
		}
	}

	// Get masternodes from the previous checkpoint header
	prevHeader := chain.GetHeaderByNumber(prevCheckpoint)
	if prevHeader == nil {
		return signers, nil
	}
	masternodes := GetMasternodesFromCheckpointHeader(prevHeader)

	// Count valid signatures per masternode
	chainConfig := chain.Config()
	for i := startBlockNumber; i <= endBlockNumber; i++ {
		if i%chainConfig.Viction.ValidatorSignInterval == 0 || !chainConfig.IsTIP2019(new(big.Int).SetUint64(i)) {
			addrs := data[mapBlkHash[i]]
			if len(addrs) > 0 {
				addrSigners := make(map[common.Address]bool)
				for _, masternode := range masternodes {
					for _, addr := range addrs {
						if addr == masternode {
							if _, ok := addrSigners[addr]; !ok {
								addrSigners[addr] = true
							}
							break
						}
					}
				}

				for addr := range addrSigners {
					if _, exist := signers[addr]; exist {
						signers[addr].Sign++
					} else {
						signers[addr] = &rewardLog{1, new(big.Int)}
					}
					*totalSigner++
				}
			}
		}
	}

	log.Info("Calculate reward at checkpoint", "startBlock", startBlockNumber, "endBlock", endBlockNumber)
	return signers, nil
}

// calculateRewardForSigner distributes reward proportionally based on signature count.
func calculateRewardForSigner(chainReward *big.Int, signers map[common.Address]*rewardLog, totalSigner uint64) map[common.Address]*big.Int {
	resultSigners := make(map[common.Address]*big.Int)
	if totalSigner > 0 {
		for signer, rLog := range signers {
			calcReward := new(big.Int)
			calcReward.Div(chainReward, new(big.Int).SetUint64(totalSigner))
			calcReward.Mul(calcReward, new(big.Int).SetUint64(rLog.Sign))
			rLog.Reward = calcReward
			resultSigners[signer] = calcReward
		}
	}
	return resultSigners
}

// getRewardBalancesRate splits reward between masternode owner, voters, and foundation.
// Owner: 40%, Voters: 50%, Foundation: 10%
func getRewardBalancesRate(foundationWalletAddr common.Address, statedb *state.StateDB, masterAddr common.Address, totalReward *big.Int, blockNumber uint64, chainConfig *params.ChainConfig) map[common.Address]*big.Int {
	validatorAddr := chainConfig.Viction.ValidatorContract
	balances := make(map[common.Address]*big.Int)

	// Masternode owner reward
	owner := state.GetCandidateOwner(statedb, validatorAddr, masterAddr)
	rewardMaster := new(big.Int).Mul(totalReward, new(big.Int).SetUint64(chainConfig.Viction.RewardValidatorPercent))
	rewardMaster = new(big.Int).Div(rewardMaster, new(big.Int).SetUint64(100))
	balances[owner] = rewardMaster

	// Voter rewards
	voters := state.GetVoters(statedb, validatorAddr, masterAddr)
	if len(voters) > 0 {
		totalVoterReward := new(big.Int).Mul(totalReward, new(big.Int).SetUint64(chainConfig.Viction.RewardVoterPercent))
		totalVoterReward = new(big.Int).Div(totalVoterReward, new(big.Int).SetUint64(100))
		totalCap := new(big.Int)

		// Get voter capacities
		voterCaps := make(map[common.Address]*big.Int)
		tip2019Block := chainConfig.TIP2019Block
		for _, voteAddr := range voters {
			if _, ok := voterCaps[voteAddr]; ok && tip2019Block != nil && tip2019Block.Uint64() <= blockNumber {
				continue // Skip duplicates after TIP2019
			}
			voterCap := state.GetVoterCap(statedb, validatorAddr, masterAddr, voteAddr)
			totalCap.Add(totalCap, voterCap)
			voterCaps[voteAddr] = voterCap
		}

		if totalCap.Cmp(new(big.Int).SetInt64(0)) > 0 {
			for addr, voteCap := range voterCaps {
				if voteCap.Cmp(new(big.Int).SetInt64(0)) > 0 {
					rcap := new(big.Int).Mul(totalVoterReward, voteCap)
					rcap = new(big.Int).Div(rcap, totalCap)
					if balances[addr] != nil {
						balances[addr].Add(balances[addr], rcap)
					} else {
						balances[addr] = rcap
					}
				}
			}
		}
	}

	// Foundation reward
	foundationReward := new(big.Int).Mul(totalReward, new(big.Int).SetUint64(chainConfig.Viction.RewardFoundationPercent))
	foundationReward = new(big.Int).Div(foundationReward, new(big.Int).SetUint64(100))
	balances[foundationWalletAddr] = foundationReward

	return balances
}

// calcInitialReward computes the VIC reward per epoch with halving schedule.
// Full reward for first 2 years, half for years 2-5, quarter for years 5-8, zero after 8 years.
func calcInitialReward(rewardPerEpoch *big.Int, number uint64, blockPerYear uint64) *big.Int {
	if blockPerYear*8 <= number {
		return big.NewInt(0)
	}
	if blockPerYear*5 <= number {
		return new(big.Int).Div(rewardPerEpoch, new(big.Int).SetUint64(4))
	}
	if blockPerYear*2 <= number {
		return new(big.Int).Div(rewardPerEpoch, new(big.Int).SetUint64(2))
	}
	return new(big.Int).Set(rewardPerEpoch)
}

// calcSaigonReward computes additional reward from Saigon upgrade with 4-year halving cycles.
func calcSaigonReward(rewardPerEpoch *big.Int, saigonBlock *big.Int, number uint64, blockPerYear uint64) *big.Int {
	if saigonBlock == nil {
		return big.NewInt(0)
	}
	headBlock := new(big.Int).SetUint64(number)
	yearsFromHardfork := new(big.Int).Div(
		new(big.Int).Sub(headBlock, saigonBlock),
		new(big.Int).SetUint64(blockPerYear),
	)
	// Additional reward lasts 16 years
	if yearsFromHardfork.Cmp(big.NewInt(0)) < 0 || yearsFromHardfork.Cmp(big.NewInt(16)) >= 0 {
		return big.NewInt(0)
	}
	cyclesFromHardfork := new(big.Int).Div(yearsFromHardfork, big.NewInt(4))
	rewardHalving := new(big.Int).Exp(big.NewInt(2), cyclesFromHardfork, nil)
	return new(big.Int).Div(rewardPerEpoch, rewardHalving)
}
