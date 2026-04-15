package ethapi

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/sortlgc"
)

const (
	// statuses of candidates
	statusMasternode = "MASTERNODE"
	statusSlashed    = "SLASHED"
	statusProposed   = "PROPOSED"
	fieldStatus      = "status"
	fieldCapacity    = "capacity"
	fieldCandidates  = "candidates"
	fieldSuccess     = "success"
	fieldEpoch       = "epoch"
)

var errEmptyHeader = errors.New("empty header")

// PublicVictionBlockChainAPI exposes Viction/PoSV-specific read-only blockchain helpers.
type PublicVictionBlockChainAPI struct {
	b BackendViction
}

func NewPublicVictionBlockChainAPI(b BackendViction) *PublicVictionBlockChainAPI {
	return &PublicVictionBlockChainAPI{b}
}

func (s *PublicVictionBlockChainAPI) GetBlockSignersByHash(ctx context.Context, blockHash common.Hash) ([]common.Address, error) {
	block, err := s.b.BlockByHash(ctx, blockHash)
	if err != nil || block == nil {
		return []common.Address{}, err
	}
	validators, err := s.GetMasternodes(ctx, block)
	if err != nil || len(validators) == 0 {
		return []common.Address{}, err
	}
	return s.rpcOutputBlockSigners(ctx, block)
}

func (s *PublicVictionBlockChainAPI) GetBlockSignersByNumber(ctx context.Context, blockNumber rpc.BlockNumber) ([]common.Address, error) {
	block, err := s.b.BlockByNumber(ctx, blockNumber)
	if err != nil || block == nil {
		return []common.Address{}, err
	}
	validators, err := s.GetMasternodes(ctx, block)
	if err != nil || len(validators) == 0 {
		return []common.Address{}, err
	}
	return s.rpcOutputBlockSigners(ctx, block)
}

func (s *PublicVictionBlockChainAPI) GetBlockFinalityByHash(ctx context.Context, blockHash common.Hash) (uint, error) {
	block, err := s.b.BlockByHash(ctx, blockHash)
	if err != nil || block == nil {
		return 0, err
	}
	validators, err := s.GetMasternodes(ctx, block)
	if err != nil || len(validators) == 0 {
		return 0, err
	}
	return s.findFinalityOfBlock(ctx, block, validators)
}

func (s *PublicVictionBlockChainAPI) GetBlockFinalityByNumber(ctx context.Context, blockNumber rpc.BlockNumber) (uint, error) {
	block, err := s.b.BlockByNumber(ctx, blockNumber)
	if err != nil || block == nil {
		return uint(0), err
	}
	validators, err := s.GetMasternodes(ctx, block)
	if err != nil || len(validators) == 0 {
		log.Error("Failed to get validators", "err", err, "len(validators)", len(validators))
		return uint(0), err
	}
	return s.findFinalityOfBlock(ctx, block, validators)
}

// GetMasternodes returns masternodes set at the starting block of epoch of the given block
func (s *PublicVictionBlockChainAPI) GetMasternodes(ctx context.Context, b *types.Block) ([]common.Address, error) {
	var masternodes []common.Address
	interval := s.b.ChainConfig().Viction.ValidatorSignInterval
	if b.Number().Int64() > 0 {
		curBlockNumber := b.Number().Uint64()
		prevBlockNumber := curBlockNumber - (interval - (curBlockNumber % interval))
		latestBlockNumber := s.b.CurrentBlock().Number()
		if prevBlockNumber >= latestBlockNumber.Uint64() || !s.b.ChainConfig().IsTIP2019(new(big.Int).SetUint64(curBlockNumber)) {
			prevBlockNumber = curBlockNumber
		}
		if _, ok := s.b.Engine().(*posv.Posv); ok {
			// Get block epoc latest.
			lastCheckpointNumber := prevBlockNumber - (prevBlockNumber % s.b.ChainConfig().Posv.Epoch)
			prevCheckpointBlock, _ := s.b.BlockByNumber(ctx, rpc.BlockNumber(lastCheckpointNumber))
			if prevCheckpointBlock != nil {
				masternodes = posv.ExtractValidatorsFromCheckpointHeader(prevCheckpointBlock.Header())
			}
		} else {
			log.Error("Engine is not a *posv.Posv")
		}
	}
	return masternodes, nil
}

// GetCandidateStatus returns status of the given candidate at a specified epochNumber
func (s *PublicVictionBlockChainAPI) GetCandidateStatus(ctx context.Context, coinbaseAddress common.Address, epoch rpc.EpochNumber) (map[string]interface{}, error) {
	var (
		block                    *types.Block
		header                   *types.Header
		checkpointNumber         rpc.BlockNumber
		epochNumber              rpc.EpochNumber // if epoch == "latest", print the latest epoch number to epochNumber
		masternodes, penaltyList []common.Address
		candidates               []posv.Masternode
		penalties                []byte
		err                      error
	)

	result := map[string]interface{}{
		fieldStatus:   "",
		fieldCapacity: 0,
		fieldSuccess:  true,
	}

	epochConfig := s.b.ChainConfig().Posv.Epoch

	// checkpoint block
	checkpointNumber, epochNumber = s.GetPreviousCheckpointFromEpoch(ctx, epoch)
	result[fieldEpoch] = epochNumber

	block, err = s.b.BlockByNumber(ctx, checkpointNumber)
	if err != nil || block == nil { // || checkpointNumber == 0 {
		result[fieldSuccess] = false
		return result, err
	}

	header = block.Header()
	if header == nil {
		log.Error("Empty header at checkpoint ", "num", checkpointNumber)
		return result, errEmptyHeader
	}
	// list of candidates (masternode, slash, propose) from either latest state
	// or checkpoint state, depending on requested epoch.
	stateBlock := checkpointNumber
	if epoch == rpc.LatestEpochNumber {
		stateBlock = rpc.BlockNumber(s.b.CurrentBlock().Number().Uint64())
	}
	statedb, _, err := s.b.StateAndHeaderByNumber(ctx, stateBlock)
	if err != nil {
		log.Error("Failed to get state and header by number", "block", stateBlock, "err", err)
		result[fieldSuccess] = false
		return result, fmt.Errorf("failed to get state and header by number %d: %w", stateBlock, err)
	}

	listOfCandidates := statedb.VicGetCandidates(s.b.ChainConfig().Viction.ValidatorContract)
	for _, candidate := range listOfCandidates {
		_, cap := statedb.VicGetValidatorInfo(s.b.ChainConfig().Viction.ValidatorContract, candidate)
		if candidate.String() != "0x0000000000000000000000000000000000000000" {
			candidates = append(candidates, posv.Masternode{Address: candidate, Stake: cap})
		}
	}

	if len(candidates) == 0 {
		err = fmt.Errorf("no masternode found")
	} else if s.b.ChainConfig().IsAtlas(header.Number) {
		sort.SliceStable(candidates, func(i, j int) bool {
			return candidates[i].Stake.Cmp(candidates[j].Stake) >= 0
		})
	} else {
		sortlgc.Slice(candidates, func(i, j int) bool {
			return candidates[i].Stake.Cmp(candidates[j].Stake) >= 0
		})
	}
	if err != nil || len(candidates) == 0 {
		log.Debug("Candidates list cannot be found", "len(candidates)", len(candidates), "err", err)
		result[fieldSuccess] = false
		return result, err
	}

	isTopCandidate := false
	// check penalties from checkpoint headers and modify status of a node to SLASHED if it's in top 150 candidates
	// if it's SLASHED but it's out of top 150, the status should be still PROPOSED
	for i := 0; i < len(candidates); i++ {
		if coinbaseAddress == candidates[i].Address {
			if i < int(s.b.ChainConfig().Viction.ValidatorMaxCount) {
				isTopCandidate = true
			}
			result[fieldStatus] = statusProposed
			result[fieldCapacity] = candidates[i].Stake
			break
		}
	}
	if !isTopCandidate {
		return result, nil
	}

	// Second, Find candidates that have masternode status
	if _, ok := s.b.Engine().(*posv.Posv); ok {
		masternodes = posv.ExtractValidatorsFromCheckpointHeader(header)
		if len(masternodes) == 0 {
			log.Error("Failed to get masternodes", "err", err, "len(masternodes)", len(masternodes), "blockNum", header.Number.Uint64())
			result[fieldSuccess] = false
			return result, err
		}
	} else {
		log.Error("Undefined POSV consensus engine")
	}
	// Set masternode status
	for _, masternode := range masternodes {
		if coinbaseAddress == masternode {
			result[fieldStatus] = statusMasternode
			return result, nil
		}
	}

	// Third, Get penalties list
	penalties = append(penalties, header.Penalties...)
	// check last 5 epochs to find penalize masternodes
	for i := 1; i <= int(s.b.ChainConfig().Viction.PenaltyEpochCount); i++ {
		if header.Number.Uint64() < epochConfig*uint64(i) {
			break
		}
		blockNum := header.Number.Uint64() - epochConfig*uint64(i)
		checkpointHeader, err := s.b.HeaderByNumber(ctx, rpc.BlockNumber(blockNum))
		if checkpointHeader == nil || err != nil {
			log.Error("Failed to get header by number", "num", blockNum, "err", err)
			continue
		}
		penalties = append(penalties, checkpointHeader.Penalties...)
	}
	penaltyList = posv.DecodePenaltiesFromHeader(penalties)

	// map slashing status
	for _, pen := range penaltyList {
		if coinbaseAddress == pen {
			result[fieldStatus] = statusSlashed
			return result, nil
		}
	}
	return result, nil
}

// GetCandidates returns status of all candidates at a specified epochNumber.
func (s *PublicVictionBlockChainAPI) GetCandidates(ctx context.Context, epoch rpc.EpochNumber) (map[string]interface{}, error) {
	var (
		block                    *types.Block
		header                   *types.Header
		checkpointNumber         rpc.BlockNumber
		epochNumber              rpc.EpochNumber
		masternodes, penaltyList []common.Address
		candidates               []posv.Masternode
		penalties                []byte
		err                      error
	)
	result := map[string]interface{}{
		fieldSuccess: true,
	}
	epochConfig := s.b.ChainConfig().Posv.Epoch
	candidatesStatusMap := map[string]map[string]interface{}{}

	checkpointNumber, epochNumber = s.GetPreviousCheckpointFromEpoch(ctx, epoch)
	result[fieldEpoch] = int64(epochNumber)

	block, err = s.b.BlockByNumber(ctx, checkpointNumber)
	if err != nil || block == nil {
		result[fieldSuccess] = false
		return result, err
	}
	header = block.Header()
	if header == nil {
		log.Error("Empty header at checkpoint", "num", checkpointNumber)
		return result, errEmptyHeader
	}

	// Read candidate list from latest or checkpoint state.
	stateBlock := checkpointNumber
	if epoch == rpc.LatestEpochNumber {
		stateBlock = rpc.BlockNumber(s.b.CurrentBlock().Number().Uint64())
	} else {
		stateBlock = rpc.BlockNumber(epochNumber)
	}
	statedb, _, err := s.b.StateAndHeaderByNumber(ctx, stateBlock)
	if err != nil {
		result[fieldSuccess] = false
		return result, err
	}
	candidateAddrs := statedb.VicGetCandidates(s.b.ChainConfig().Viction.ValidatorContract)
	for _, addr := range candidateAddrs {
		if addr == (common.Address{}) {
			continue
		}
		_, cap := statedb.VicGetValidatorInfo(s.b.ChainConfig().Viction.ValidatorContract, addr)
		candidates = append(candidates, posv.Masternode{Address: addr, Stake: cap})
	}
	if len(candidates) == 0 {
		result[fieldSuccess] = false
		return result, fmt.Errorf("no masternode found")
	}
	if s.b.ChainConfig().IsAtlas(header.Number) {
		sort.SliceStable(candidates, func(i, j int) bool {
			return candidates[i].Stake.Cmp(candidates[j].Stake) >= 0
		})
	} else {
		sortlgc.Slice(candidates, func(i, j int) bool {
			return candidates[i].Stake.Cmp(candidates[j].Stake) >= 0
		})
	}

	// First, set all candidates to PROPOSED.
	for _, candidate := range candidates {
		candidatesStatusMap[candidate.Address.String()] = map[string]interface{}{
			fieldStatus:   statusProposed,
			fieldCapacity: candidate.Stake,
		}
	}

	// Second, mark masternodes.
	if _, ok := s.b.Engine().(*posv.Posv); ok {
		masternodes = posv.ExtractValidatorsFromCheckpointHeader(header)
		if len(masternodes) == 0 {
			result[fieldSuccess] = false
			return result, fmt.Errorf("failed to get masternodes at checkpoint %d", header.Number.Uint64())
		}
	} else {
		log.Error("Undefined POSV consensus engine")
	}
	for _, masternode := range masternodes {
		if candidatesStatusMap[masternode.String()] != nil {
			candidatesStatusMap[masternode.String()][fieldStatus] = statusMasternode
		}
	}

	// Third, collect penalties and mark top-N candidates as slashed.
	penalties = append(penalties, header.Penalties...)
	for i := 1; i <= int(s.b.ChainConfig().Viction.PenaltyEpochCount); i++ {
		if header.Number.Uint64() < epochConfig*uint64(i) {
			break
		}
		blockNum := header.Number.Uint64() - epochConfig*uint64(i)
		checkpointHeader, err := s.b.HeaderByNumber(ctx, rpc.BlockNumber(blockNum))
		if checkpointHeader == nil || err != nil {
			log.Error("Failed to get header by number", "num", blockNum, "err", err)
			continue
		}
		penalties = append(penalties, checkpointHeader.Penalties...)
	}
	if len(penalties) > 0 {
		penaltyList = posv.DecodePenaltiesFromHeader(penalties)
		topCandidates := candidates
		maxCount := int(s.b.ChainConfig().Viction.ValidatorMaxCount)
		if len(topCandidates) > maxCount {
			topCandidates = topCandidates[:maxCount]
		}
		topSet := make(map[common.Address]struct{}, len(topCandidates))
		for _, c := range topCandidates {
			topSet[c.Address] = struct{}{}
		}
		for _, pen := range penaltyList {
			if _, ok := topSet[pen]; ok && candidatesStatusMap[pen.String()] != nil {
				candidatesStatusMap[pen.String()][fieldStatus] = statusSlashed
			}
		}
	}

	result[fieldCandidates] = candidatesStatusMap
	return result, nil
}

// GetPreviousCheckpointFromEpoch returns header of the previous checkpoint
func (s *PublicVictionBlockChainAPI) GetPreviousCheckpointFromEpoch(ctx context.Context, epochNum rpc.EpochNumber) (rpc.BlockNumber, rpc.EpochNumber) {
	var checkpointNumber uint64
	epoch := s.b.ChainConfig().Posv.Epoch

	if epochNum == rpc.LatestEpochNumber {
		blockNumer := s.b.CurrentBlock().Number().Uint64()
		diff := blockNumer % epoch
		// checkpoint number
		checkpointNumber = blockNumer - diff
		epochNum = rpc.EpochNumber(checkpointNumber / epoch)
		if diff > 0 {
			epochNum += 1
		}
	} else if epochNum < 2 {
		checkpointNumber = 0
	} else {
		checkpointNumber = epoch * (uint64(epochNum) - 1)
	}
	return rpc.BlockNumber(checkpointNumber), epochNum
}

func (s *PublicVictionBlockChainAPI) rpcOutputBlockSigners(ctx context.Context, b *types.Block) ([]common.Address, error) {
	engine, ok := s.b.Engine().(*posv.Posv)
	if !ok {
		log.Error("Undefined POSV consensus engine")
		return []common.Address{}, nil
	}

	signedBlock := s.findNearestSignedBlock(ctx, b)
	if signedBlock == nil {
		return []common.Address{}, nil
	}
	return s.getSigners(ctx, signedBlock, engine)
}

// findNearestSignedBlock finds the nearest checkpoint from input block
func (s *PublicVictionBlockChainAPI) findNearestSignedBlock(ctx context.Context, b *types.Block) *types.Block {
	if b.Number().Int64() <= 0 {
		return nil
	}
	interval := s.b.ChainConfig().Viction.ValidatorSignInterval
	blockNumber := b.Number().Uint64()
	signedBlockNumber := blockNumber + (interval - (blockNumber % interval))
	latestBlockNumber := s.b.CurrentBlock().Number()

	if signedBlockNumber >= latestBlockNumber.Uint64() || !s.b.ChainConfig().IsTIP2019(new(big.Int).SetUint64(signedBlockNumber)) {
		signedBlockNumber = blockNumber
	}

	checkpointBlockNumber := signedBlockNumber - (signedBlockNumber % s.b.ChainConfig().Posv.Epoch)
	checkpointBlock, _ := s.b.BlockByNumber(ctx, rpc.BlockNumber(checkpointBlockNumber))

	if checkpointBlock != nil {
		signedBlock, _ := s.b.BlockByNumber(ctx, rpc.BlockNumber(signedBlockNumber))
		return signedBlock
	}
	return nil
}

func (s *PublicVictionBlockChainAPI) getSigners(ctx context.Context, block *types.Block, engine *posv.Posv) ([]common.Address, error) {
	var filterSigners []common.Address
	var signers []common.Address
	blockNumber := block.Number().Uint64()

	checkpointBlockNumber := blockNumber - (blockNumber % s.b.ChainConfig().Posv.Epoch)
	checkpointBlock, _ := s.b.BlockByNumber(ctx, rpc.BlockNumber(checkpointBlockNumber))
	if checkpointBlock == nil {
		return nil, fmt.Errorf("checkpoint block not found at %d", checkpointBlockNumber)
	}

	validators := posv.ExtractValidatorsFromCheckpointHeader(checkpointBlock.Header())
	var err error
	signers, err = GetSignersFromBlocks(s.b, block.NumberU64(), block.Hash(), validators)
	if err != nil {
		log.Error("getSigners: failed to get signers from blocks", "err", err)
		return nil, err
	}
	attestor, _ := engine.Attestor(block.Header())
	creator, _ := engine.Author(block.Header())
	signers = append(signers, attestor, creator)

	for _, validator := range validators {
		for _, signer := range signers {
			if signer == validator {
				filterSigners = append(filterSigners, signer)
				break
			}
		}
	}
	return filterSigners, nil
}

// backendWithBlockChain is implemented by *eth.EthAPIBackend for full nodes.
type backendWithBlockChain interface {
	Backend
	BlockChain() consensus.ChainReader
}

func GetSignersFromBlocks(b Backend, blockNumber uint64, blockHash common.Hash, masternodes []common.Address) ([]common.Address, error) {
	ctx := context.Background()
	var addrs []common.Address
	mapMN := map[common.Address]bool{}
	for _, node := range masternodes {
		mapMN[node] = true
	}

	signer := types.MakeSigner(b.ChainConfig(), new(big.Int).SetUint64(blockNumber))
	if engine, ok := b.Engine().(*posv.Posv); ok {
		limitNumber := blockNumber + b.ChainConfig().Viction.LimitTimeFinality
		currentNumber := b.CurrentBlock().Number()
		if limitNumber > currentNumber.Uint64() {
			limitNumber = currentNumber.Uint64()
		}
		p, ok := b.(backendWithBlockChain)
		if !ok {
			return addrs, fmt.Errorf("backend does not expose BlockChain (need full node EthAPIBackend)")
		}
		chain := p.BlockChain()
		for i := blockNumber + 1; i <= limitNumber; i++ {
			header, err := b.HeaderByNumber(ctx, rpc.BlockNumber(i))
			if err != nil {
				return addrs, err
			}
			signTxs, err := engine.GetSignDataForBlock(b.ChainConfig(), b.ChainConfig().Viction, header, chain)
			if err != nil {
				continue
			}
			for _, signTx := range signTxs {
				blkHash := common.BytesToHash(signTx.Data()[len(signTx.Data())-32:])
				from, _ := types.Sender(signer, &signTx)
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

// findFinalityOfBlock returns finality depth using block hash cache at the signed-block level.
func (s *PublicVictionBlockChainAPI) findFinalityOfBlock(ctx context.Context, b *types.Block, masternodes []common.Address) (uint, error) {
	engine, ok := s.b.Engine().(*posv.Posv)
	if !ok {
		log.Error("Undefined POSV consensus engine")
		return 0, nil
	}
	signedBlock := s.findNearestSignedBlock(ctx, b)
	if signedBlock == nil {
		return 0, nil
	}
	signedBlocksHash := s.b.GetBlocksHashCache(signedBlock.Number().Uint64())

	// there is no cache for this block's number
	// return the number(signers) / number(masternode) * 100 if this block is on canonical path
	// else return 0 for fork path
	if signedBlocksHash == nil {
		if !s.b.AreTwoBlockSamePath(signedBlock.Hash(), b.Hash()) {
			return 0, nil
		}
		blockSigners, err := s.getSigners(ctx, signedBlock, engine)
		if blockSigners == nil {
			return 0, err
		}

		return uint(100 * len(blockSigners) / len(masternodes)), nil

	}

	/*
		With Hashes cache - we can track all chain's path
		back to current's block number by parent's Hash
		If found the current block so the finality = signedBlock's finality
		else return 0
	*/
	var signedBlockSamePath common.Hash

	for count := 0; count < len(signedBlocksHash); count++ {
		blockHash := signedBlocksHash[count]
		if s.b.AreTwoBlockSamePath(blockHash, b.Hash()) {
			signedBlockSamePath = blockHash
			break
		}
	}

	// return 0 if not same path with any signed block
	if len(signedBlockSamePath) == 0 {
		return 0, nil
	}

	// get signers and return finality
	samePathSignedBlock, err := s.b.BlockByHash(ctx, signedBlockSamePath)
	if samePathSignedBlock == nil {
		return 0, err
	}

	blockSigners, err := s.getSigners(ctx, samePathSignedBlock, engine)
	if blockSigners == nil {
		return 0, err
	}

	return uint(100 * len(blockSigners) / len(masternodes)), nil

}

// GetRewardByHash returns epoch reward data for a checkpoint block hash.
func (s *PublicVictionBlockChainAPI) GetRewardByHash(hash common.Hash) *posv.EpochReward {
	return s.b.GetRewardByHash(hash)
}

func (s *PublicVictionBlockChainAPI) GetVotersRewards(masternodeAddr common.Address, blockHash common.Hash) map[common.Address]*big.Int {
	return s.b.GetVotersRewards(masternodeAddr, blockHash)
}

func (s *PublicVictionBlockChainAPI) GetEpochDuration() *big.Int {
	return s.b.GetEpochDuration()
}

func (s *PublicVictionBlockChainAPI) GetMasternodesCap(checkpoint uint64) map[common.Address]*big.Int {
	return s.b.GetMasternodesCap(checkpoint)
}

func (s *PublicVictionBlockChainAPI) GetBlocksHashCache(blockNr uint64) []common.Hash {
	return s.b.GetBlocksHashCache(blockNr)
}

func (s *PublicVictionBlockChainAPI) AreTwoBlockSamePath(bh1 common.Hash, bh2 common.Hash) bool {
	return s.b.AreTwoBlockSamePath(bh1, bh2)
}
