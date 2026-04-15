package ethapi

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/rpc"
)

// PublicVictionBlockChainAPI exposes Viction/PoSV-specific read-only blockchain helpers.
type PublicVictionBlockChainAPI struct {
	b BackendViction
}

func NewPublicVictionBlockChainAPI(b BackendViction) *PublicVictionBlockChainAPI {
	return &PublicVictionBlockChainAPI{b}
}

// GetAttestorsPairsByHash returns attestors pairs for a checkpoint block hash.
func (s *PublicVictionBlockChainAPI) GetAttestorsPairsByHash(hash common.Hash) (map[common.Address]common.Address, error) {
	return s.b.GetAttestorsPairsByHash(hash)
}

// GetAttestorsPairsByNumber returns attestors pairs for a checkpoint block number.
func (s *PublicVictionBlockChainAPI) GetAttestorsPairsByNumber(number rpc.BlockNumber) (map[common.Address]common.Address, error) {
	return s.b.GetAttestorsPairsByNumber(number)
}

func (s *PublicVictionBlockChainAPI) GetRewardByHash(hash common.Hash) (*posv.EpochReward, error) {
	return s.b.GetRewardByHash(hash)
}

func (s *PublicVictionBlockChainAPI) GetAttestorsByHashAtCheckPoint(hash common.Hash) ([]int64, error) {
	return s.b.GetAttestorsByHashAtCheckPoint(hash)
}

func (s *PublicVictionBlockChainAPI) GetAttestorsByNumberAtCheckPoint(number rpc.BlockNumber) ([]int64, error) {
	return s.b.GetAttestorsByNumberAtCheckPoint(number)
}

func (s *PublicVictionBlockChainAPI) GetPenaltiesByHashAtCheckPoint(hash common.Hash) ([]common.Address, error) {
	return s.b.GetPenaltiesByHashAtCheckPoint(hash)
}

func (s *PublicVictionBlockChainAPI) GetPenaltiesByNumberAtCheckPoint(number rpc.BlockNumber) ([]common.Address, error) {
	return s.b.GetPenaltiesByNumberAtCheckPoint(number)
}
