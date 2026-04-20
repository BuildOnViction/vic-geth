// Copyright 2026 The Vic-geth Authors
package eth

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

func (s *EthAPIBackend) GetRewardByHash(hash common.Hash) (*posv.EpochReward, error) {
	header := s.eth.blockchain.GetHeaderByHash(hash)
	if header == nil || header.Number.Uint64()%s.eth.blockchain.Config().Posv.Epoch != 0 {
		return nil, errors.New("header is not a checkpoint block")
	}
	engine := s.Engine().(*posv.Posv)
	statedb, err := s.eth.blockchain.StateAt(header.Root)
	if err != nil {
		log.Info("Failed to get state at", "hash", hash, "error", err)
		return nil, err
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
		return nil, err
	}
	if epochReward == nil {
		return nil, errors.New("epoch reward is nil")
	}
	return epochReward, nil

}

func (s *EthAPIBackend) GetAttestorsPairsByHash(hash common.Hash) (map[common.Address]common.Address, error) {
	posvConfig := s.eth.blockchain.Config().Posv
	header := s.eth.blockchain.GetHeaderByHash(hash)
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

func (s *EthAPIBackend) GetAttestorsPairsByNumber(number rpc.BlockNumber) (map[common.Address]common.Address, error) {
	var blockNumber uint64
	switch number {
	case rpc.LatestBlockNumber:
		blockNumber = s.CurrentBlock().NumberU64()
	case rpc.PendingBlockNumber:
		return nil, errors.New("pending block number is not supported")
	default:
		if number < 0 {
			return nil, errors.New("invalid block number")
		}
		blockNumber = uint64(number)
	}

	posvConfig := s.eth.blockchain.Config().Posv
	header := s.eth.blockchain.GetHeaderByNumber(blockNumber)
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

func (s *EthAPIBackend) GetAttestorsByHashAtCheckPoint(hash common.Hash) ([]int64, error) {
	header := s.eth.blockchain.GetHeaderByHash(hash)
	if header == nil || header.Number.Uint64()%s.eth.blockchain.Config().Posv.Epoch != 0 {
		return nil, errors.New("header is not a checkpoint block")
	}
	_, ok := s.Engine().(*posv.Posv)
	if !ok {
		return nil, errors.New("engine is not a posv engine")
	}
	attestors := posv.ExtractAttestorsFromCheckpointHeader(header)
	return attestors, nil
}

func (s *EthAPIBackend) GetAttestorsByNumberAtCheckPoint(number rpc.BlockNumber) ([]int64, error) {
	var blockNumber uint64
	switch number {
	case rpc.LatestBlockNumber:
		blockNumber = s.CurrentBlock().NumberU64()
	case rpc.PendingBlockNumber:
		return nil, errors.New("pending block number is not supported")
	default:
		if number < 0 {
			return nil, errors.New("invalid block number")
		}
		blockNumber = uint64(number)
	}
	header := s.eth.blockchain.GetHeaderByNumber(blockNumber)
	if header == nil || header.Number.Uint64()%s.eth.blockchain.Config().Posv.Epoch != 0 {
		return nil, errors.New("header not found")
	}
	_, ok := s.Engine().(*posv.Posv)
	if !ok {
		return nil, errors.New("engine is not a posv engine")
	}
	attestors := posv.ExtractAttestorsFromCheckpointHeader(header)
	return attestors, nil
}

func (s *EthAPIBackend) GetPenaltiesByHashAtCheckPoint(hash common.Hash) ([]common.Address, error) {
	header := s.eth.blockchain.GetHeaderByHash(hash)
	if header == nil || header.Number.Uint64()%s.eth.blockchain.Config().Posv.Epoch != 0 {
		return nil, errors.New("header is not a checkpoint block")
	}
	_, ok := s.Engine().(*posv.Posv)
	if !ok {
		return nil, errors.New("engine is not a posv engine")
	}

	penalties := posv.DecodePenaltiesFromHeader(header.Penalties)
	return penalties, nil
}

func (s *EthAPIBackend) GetPenaltiesByNumberAtCheckPoint(number rpc.BlockNumber) ([]common.Address, error) {
	var blockNumber uint64
	switch number {
	case rpc.LatestBlockNumber:
		blockNumber = s.CurrentBlock().NumberU64()
	case rpc.PendingBlockNumber:
		return nil, errors.New("pending block number is not supported")
	default:
		if number < 0 {
			return nil, errors.New("invalid block number")
		}
		blockNumber = uint64(number)
	}
	header := s.eth.blockchain.GetHeaderByNumber(blockNumber)
	if header == nil || header.Number.Uint64()%s.eth.blockchain.Config().Posv.Epoch != 0 {
		return nil, errors.New("header is not a checkpoint block")
	}
	_, ok := s.Engine().(*posv.Posv)
	if !ok {
		return nil, errors.New("engine is not a posv engine")
	}

	penalties := posv.DecodePenaltiesFromHeader(header.Penalties)
	return penalties, nil
}
