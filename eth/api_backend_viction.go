// Copyright 2026 The Vic-geth Authors
package eth

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// GetRewardByHash returns the epoch reward breakdown for the checkpoint block identified by hash.
// The hash must identify a checkpoint block (block number divisible by the epoch size, e.g. 900,
// 1800, 2700 ...). Passing a non-checkpoint hash returns an error.
//
// Note: requires the state trie at that block to be available. On a pruning (full-sync) node,
// state is only kept for the most recent ~128 blocks; querying an older checkpoint will fail
// with "missing trie node" unless the node runs in archive mode (--gcmode=archive).
func (s *EthAPIBackend) GetRewardByHash(ctx context.Context, hash common.Hash) (*posv.EpochReward, error) {
	header, err := s.HeaderByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return s.getEpochRewardByCheckpointHeader(header)
}

// GetRewardByNumber returns the epoch reward breakdown for the checkpoint block at the given number.
// The block number must be divisible by the epoch size (e.g. 900, 1800, 2700 ...).
// Passing a non-checkpoint number returns an error.
//
// Note: requires the state trie at that block to be available. On a pruning (full-sync) node,
// state is only kept for the most recent ~128 blocks; querying an older checkpoint will fail
// with "missing trie node" unless the node runs in archive mode (--gcmode=archive).
func (s *EthAPIBackend) GetRewardByNumber(ctx context.Context, number rpc.BlockNumber) (*posv.EpochReward, error) {
	header, err := s.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	return s.getEpochRewardByCheckpointHeader(header)
}

// GetRewardByHashOrNumber returns the epoch reward breakdown for the checkpoint block identified
// by either a block hash or a block number. The resolved block must be a checkpoint block
// (block number divisible by the epoch size). Passing a non-checkpoint identifier returns an error.
//
// Note: requires the state trie at that block to be available. On a pruning (full-sync) node,
// state is only kept for the most recent ~128 blocks; querying an older checkpoint will fail
// with "missing trie node" unless the node runs in archive mode (--gcmode=archive).
func (s *EthAPIBackend) GetRewardByHashOrNumber(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*posv.EpochReward, error) {
	header, err := s.HeaderByNumberOrHash(ctx, blockNrOrHash)
	if err != nil {
		return nil, err
	}
	return s.getEpochRewardByCheckpointHeader(header)
}

func (s *EthAPIBackend) getEpochRewardByCheckpointHeader(header *types.Header) (*posv.EpochReward, error) {
	if header == nil {
		return nil, errors.New("header at block number or hash is not found")
	}

	cfg := s.eth.blockchain.Config()
	if header.Number.Uint64()%cfg.Posv.Epoch != 0 {
		return nil, errors.New("header is not a checkpoint block")
	}

	engine, ok := s.Engine().(*posv.Posv)
	if !ok {
		return nil, errors.New("engine is not a posv engine")
	}

	statedb, err := s.eth.blockchain.StateAt(header.Root)
	if err != nil {
		log.Info("Failed to get state at", "hash", header.Hash(), "error", err)
		return nil, err
	}

	epochReward, err := s.eth.PosvGetEpochReward(
		engine,
		cfg,
		cfg.Posv,
		cfg.Viction,
		header,
		s.eth.blockchain,
		statedb,
		log.New(),
	)
	if err != nil {
		return nil, err
	}
	if epochReward == nil {
		return nil, errors.New("epoch reward is nil")
	}
	return epochReward, nil
}

func (s *EthAPIBackend) GetAttestorsPairsByHash(ctx context.Context, hash common.Hash) (map[common.Address]common.Address, error) {
	header, err := s.HeaderByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if header == nil {
		return nil, errors.New("header not found")
	}
	posvConfig := s.eth.blockchain.Config().Posv
	checkpointHeader := posv.GetCheckpointHeader(posvConfig, header, s.eth.blockchain, nil)
	if checkpointHeader == nil {
		return nil, errors.New("checkpoint header not found")
	}
	engine, ok := s.Engine().(*posv.Posv)
	if !ok {
		return nil, errors.New("engine is not a posv engine")
	}
	pairs, _, err := s.eth.PosvGetCreatorAttestorPairs(engine, s.eth.blockchain.Config(), header, checkpointHeader)
	return pairs, err
}

func (s *EthAPIBackend) GetAttestorsPairsByNumber(ctx context.Context, number rpc.BlockNumber) (map[common.Address]common.Address, error) {
	header, err := s.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, err
	}

	posvConfig := s.eth.blockchain.Config().Posv
	if header == nil {
		return nil, errors.New("header not found")
	}
	checkpointHeader := posv.GetCheckpointHeader(posvConfig, header, s.eth.blockchain, nil)
	if checkpointHeader == nil {
		return nil, errors.New("checkpoint header not found")
	}
	engine, ok := s.Engine().(*posv.Posv)
	if !ok {
		return nil, errors.New("engine is not a posv engine")
	}
	pairs, _, err := s.eth.PosvGetCreatorAttestorPairs(engine, s.eth.blockchain.Config(), header, checkpointHeader)
	return pairs, err
}

func (s *EthAPIBackend) GetBlockFinalityByHash(ctx context.Context, blockHash common.Hash) (uint, error) {
	block, err := s.BlockByHash(ctx, blockHash)
	if err != nil || block == nil {
		return uint(0), err
	}
	return s.getBlockFinality(ctx, block)

}

func (s *EthAPIBackend) GetBlockFinalityByNumber(ctx context.Context, blockNumber rpc.BlockNumber) (uint, error) {
	block, err := s.BlockByNumber(ctx, rpc.BlockNumber(blockNumber))
	if err != nil || block == nil {
		return uint(0), err
	}
	return s.getBlockFinality(ctx, block)

}

// getBlockFinality computes the finality percentage for an already-fetched block.
// It first attempts the legacy per-block finality calculation, then falls back to
// scanning forward with findClosestFinalizedBlock when the legacy path fails or
// returns a partial result.
func (s *EthAPIBackend) getBlockFinality(ctx context.Context, block *types.Block) (uint, error) {
	if block.NumberU64() == 0 {
		return 100, nil
	}

	var finalityLegacy uint
	masternodes, err := s.GetMasternodes(ctx, block)
	if err != nil {
		log.Error("Failed to get masternodes", "number", block.NumberU64(), "hash", block.Hash(), "err", err)
	} else if len(masternodes) == 0 {
		log.Error("Empty masternodes", "number", block.NumberU64(), "hash", block.Hash())
	} else {
		// Masternodes available — compute legacy finality.
		finalityLegacy, err = s.findFinalityOfBlock(ctx, block, masternodes)
		if err != nil {
			// Legacy finality calculation failed; fall through to findClosestFinalizedBlock.
			log.Error("Failed to find legacy finality of block", "number", block.NumberU64(), "hash", block.Hash(), "err", err)
		} else if finalityLegacy == 100 {
			return 100, nil
		}
	}

	// findClosestFinalizedBlock scans forward from block.NumberU64(), so closest
	// (if non-nil) is always at a number >= block.NumberU64(); the same-path check
	// is the only condition that can distinguish canonical from fork blocks here.
	closest, closestFinality := s.findClosestFinalizedBlock(ctx, block.NumberU64())
	if closest == nil {
		// No finalized block found ahead; fall back to the legacy percentage.
		return finalityLegacy, nil
	}
	samePath := s.AreTwoBlockSamePath(closest.Hash(), block.Hash())
	if samePath {
		// Block is on the same canonical path as the nearest finalized block.
		if closestFinality > finalityLegacy {
			return closestFinality, nil
		}
		return finalityLegacy, nil
	}

	// closest is on a different fork — the queried block has been orphaned.
	return finalityLegacy, nil
}

// findClosestFinalizedBlock scans forward from targetNumber to the current chain
// head looking for the block with the highest finality percentage.
//
// When the TIPSigning fork is active, only MergeSignRange checkpoint blocks are
// checked (rounded up from targetNumber); otherwise every block is checked.
//
// Returns the best block found and its finality percentage (0–100).
// Returns (nil, 0) when no candidate could be evaluated (e.g. no chain head).
func (s *EthAPIBackend) findClosestFinalizedBlock(ctx context.Context, targetNumber uint64) (*types.Block, uint) {
	head := s.CurrentBlock()
	if head == nil {
		return nil, 0
	}

	headNumber := head.NumberU64()
	interval := s.eth.blockchain.Config().Viction.ValidatorSignInterval

	// Determine scan step and start position.
	// Under TIPSigning, signatures are aggregated at MergeSignRange checkpoints,
	// so only those block numbers carry meaningful finality data.
	step := uint64(1)
	scanStart := targetNumber
	if chainConfig := s.eth.blockchain.Config(); chainConfig != nil && chainConfig.IsTIPSigning(new(big.Int).SetUint64(headNumber)) {
		step = uint64(interval)
		// Round up to the first checkpoint >= targetNumber.
		if rem := targetNumber % step; rem != 0 {
			scanStart = targetNumber + (step - rem)
		}
	}

	// Track the highest-finality block seen so far.
	// Used as the fallback result when no block reaches 100%.
	var (
		bestFinality uint
		bestBlock    *types.Block
	)

	// evalBlock computes the finality of the block at number and updates the
	// running best. Returns (block, true) when 100% finality is confirmed so
	// the caller can stop early; returns (nil, false) when the block should be
	// skipped (fetch error, no masternodes, or finality error).
	evalBlock := func(number uint64) (*types.Block, bool) {
		candidate, err := s.BlockByNumber(ctx, rpc.BlockNumber(number))
		if err != nil || candidate == nil {
			return nil, false
		}
		masternodes, err := s.GetMasternodes(ctx, candidate)
		if err != nil || len(masternodes) == 0 {
			return nil, false
		}
		finality, err := s.findFinalityOfBlock(ctx, candidate, masternodes)
		if err != nil {
			return nil, false
		}
		if finality > bestFinality {
			bestFinality = finality
			bestBlock = candidate
		}
		if finality == 100 {
			return candidate, true
		}
		return nil, false
	}

	for number := scanStart; number <= headNumber; number += step {
		if candidate, found := evalBlock(number); found {
			return candidate, 100
		}
	}

	// No block reached 100%; return the best partial result if any.
	if bestFinality > 0 && bestBlock != nil {
		return bestBlock, bestFinality
	}
	return nil, 0
}

// GetMasternodes returns masternodes set at the starting block of epoch of the given block
func (s *EthAPIBackend) GetMasternodes(ctx context.Context, b *types.Block) ([]common.Address, error) {
	var masternodes []common.Address
	interval := s.eth.blockchain.Config().Viction.ValidatorSignInterval
	if b.Number().Int64() >= 0 {
		curBlockNumber := b.Number().Uint64()
		prevBlockNumber := curBlockNumber + (interval - (curBlockNumber % interval))
		latestBlockNumber := s.eth.blockchain.CurrentBlock().Number().Uint64()
		if prevBlockNumber >= latestBlockNumber || !s.eth.blockchain.Config().IsTIP2019(b.Number()) {
			prevBlockNumber = curBlockNumber
		}
		if _, ok := s.Engine().(*posv.Posv); ok {
			// Get block epoc latest.
			lastCheckpointNumber := prevBlockNumber - (prevBlockNumber % s.eth.blockchain.Config().Posv.Epoch)
			prevCheckpointBlock, _ := s.BlockByNumber(ctx, rpc.BlockNumber(lastCheckpointNumber))
			if prevCheckpointBlock != nil {
				masternodes = posv.ExtractValidatorsFromCheckpointHeader(prevCheckpointBlock.Header())
			}
		} else {
			log.Error("Undefined POSV consensus engine")
		}
	}
	return masternodes, nil
}

/*
findFinalityOfBlock return finality of a block
*/
func (s *EthAPIBackend) findFinalityOfBlock(ctx context.Context, b *types.Block, masternodes []common.Address) (uint, error) {
	engine, _ := s.Engine().(*posv.Posv)
	signedBlock := s.findNearestSignedBlock(ctx, b)

	if signedBlock == nil {
		return 0, nil
	}

	samePath := s.AreTwoBlockSamePath(signedBlock.Hash(), b.Hash())
	if !samePath {
		return 0, nil
	}

	blockSigners, err := s.getSigners(ctx, signedBlock, engine)
	if blockSigners == nil {
		return 0, err
	}

	return uint(100 * len(blockSigners) / len(masternodes)), nil

}

/*
Extract signers from block
*/
func (s *EthAPIBackend) getSigners(ctx context.Context, block *types.Block, engine *posv.Posv) ([]common.Address, error) {
	var err error
	var filterSigners []common.Address
	var signers []common.Address
	blockNumber := block.Number().Uint64()

	// Get block epoc latest.
	checkpointNumber := blockNumber - (blockNumber % s.eth.blockchain.Config().Posv.Epoch)
	checkpointBlock, _ := s.BlockByNumber(ctx, rpc.BlockNumber(checkpointNumber))

	masternodes := posv.ExtractValidatorsFromCheckpointHeader(checkpointBlock.Header())
	signers, err = s.GetSignersFromBlocks(ctx, block.NumberU64(), block.Hash(), masternodes)
	if err != nil {
		log.Error("Fail to get signers from block signer SC.", "error", err)
		return nil, err
	}
	validator, _ := engine.Attestor(block.Header())
	creator, _ := engine.Author(block.Header())
	signers = append(signers, validator)
	signers = append(signers, creator)

	for _, masternode := range masternodes {
		for _, signer := range signers {
			if signer == masternode {
				filterSigners = append(filterSigners, masternode)
				break
			}
		}
	}
	return filterSigners, nil
}

func (s *EthAPIBackend) GetSignersFromBlocks(ctx context.Context, blockNumber uint64, blockHash common.Hash, masternodes []common.Address) ([]common.Address, error) {
	var addrs []common.Address
	mapMN := map[common.Address]bool{}
	for _, node := range masternodes {
		mapMN[node] = true
	}
	signer := types.MakeSigner(s.eth.blockchain.Config(), new(big.Int).SetUint64(blockNumber))
	if engine, ok := s.Engine().(*posv.Posv); ok {
		limitNumber := blockNumber + s.eth.blockchain.Config().Viction.LimitTimeFinality
		currentNumber := s.CurrentBlock().NumberU64()
		if limitNumber > currentNumber {
			limitNumber = currentNumber
		}
		for i := blockNumber + 1; i <= limitNumber; i++ {
			header, err := s.HeaderByNumber(ctx, rpc.BlockNumber(i))
			if err != nil {
				return addrs, err
			}
			signTxs, err := engine.GetSignDataForBlock(s.eth.blockchain.Config(), s.eth.blockchain.Config().Viction, header, s.eth.blockchain)
			if err != nil {
				return addrs, err
			}
			for _, signtx := range signTxs {
				blkHash := common.BytesToHash(signtx.Data()[len(signtx.Data())-32:])
				from, _ := types.Sender(signer, &signtx)
				if blkHash == blockHash && mapMN[from] {
					addrs = append(addrs, from)
					delete(mapMN, from)
				}
			}
			if len(mapMN) == 0 {
				break
			}
		}
	}
	return addrs, nil
}

// AreTwoBlockSamePath reports whether two blocks lie on the same canonical ancestry line.
func (s *EthAPIBackend) AreTwoBlockSamePath(hash1, hash2 common.Hash) bool {
	return s.eth.blockchain.AreTwoBlockSamePath(hash1, hash2)
}

// GetValidators returns validators set at the starting block of epoch of the given block
func (s *EthAPIBackend) GetValidators(b *types.Block) ([]common.Address, error) {
	var validators []common.Address
	interval := s.eth.blockchain.Config().Viction.ValidatorSignInterval
	if b.Number().Int64() >= 0 {
		curBlockNumber := b.Number().Uint64()
		prevBlockNumber := curBlockNumber + (interval - (curBlockNumber % interval))
		latestBlockNumber := s.CurrentBlock().Number().Uint64()
		if prevBlockNumber >= latestBlockNumber || !s.eth.blockchain.Config().IsTIP2019(b.Number()) {
			prevBlockNumber = curBlockNumber
		}
		if _, ok := s.Engine().(*posv.Posv); ok {
			// Get block epoc latest.
			lastCheckpointNumber := prevBlockNumber - (prevBlockNumber % s.eth.blockchain.Config().Posv.Epoch)
			prevCheckpointBlock := s.eth.blockchain.GetBlockByNumber(lastCheckpointNumber)
			if prevCheckpointBlock != nil {
				validators = posv.ExtractValidatorsFromCheckpointHeader(prevCheckpointBlock.Header())
			}
		} else {
			log.Error("Undefined POSV consensus engine")
		}
	}
	return validators, nil
}

// findNearestSignedBlock finds the nearest checkpoint from input block
func (s *EthAPIBackend) findNearestSignedBlock(ctx context.Context, b *types.Block) *types.Block {
	if b.Number().Int64() <= 0 {
		return nil
	}
	interval := s.eth.blockchain.Config().Viction.ValidatorSignInterval
	blockNumber := b.Number().Uint64()
	signedBlockNumber := blockNumber + (interval - (blockNumber % interval))
	latestBlockNumber := s.CurrentBlock().NumberU64()

	if signedBlockNumber >= latestBlockNumber || !s.eth.blockchain.Config().IsTIPSigning(b.Number()) {
		signedBlockNumber = blockNumber
	}

	// Get block epoc latest
	checkpointNumber := signedBlockNumber - (signedBlockNumber % s.eth.blockchain.Config().Posv.Epoch)
	checkpointBlock, _ := s.BlockByNumber(ctx, rpc.BlockNumber(checkpointNumber))
	if checkpointBlock != nil {
		signedBlock := s.eth.blockchain.GetBlockByNumber(signedBlockNumber)
		return signedBlock
	}
	return nil

}
