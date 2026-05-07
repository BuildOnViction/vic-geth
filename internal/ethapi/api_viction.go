package ethapi

import (
	"context"

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
func (s *PublicVictionBlockChainAPI) GetAttestorsPairsByHash(ctx context.Context, hash common.Hash) (map[common.Address]common.Address, error) {
	return s.b.GetAttestorsPairsByHash(ctx, hash)
}

// GetAttestorsPairsByNumber returns attestors pairs for a checkpoint block number.
func (s *PublicVictionBlockChainAPI) GetAttestorsPairsByNumber(ctx context.Context, number rpc.BlockNumber) (map[common.Address]common.Address, error) {
	return s.b.GetAttestorsPairsByNumber(ctx, number)
}

func (s *PublicVictionBlockChainAPI) GetRewardByHash(ctx context.Context, hash common.Hash) (*posv.EpochReward, error) {
	return s.b.GetRewardByHash(ctx, hash)
}

func (s *PublicVictionBlockChainAPI) GetBlockFinalityByHash(ctx context.Context, hash common.Hash) (uint, error) {
	return s.b.GetBlockFinalityByHash(ctx, hash)
}

func (s *PublicVictionBlockChainAPI) GetBlockFinalityByNumber(ctx context.Context, number rpc.BlockNumber) (uint, error) {
	return s.b.GetBlockFinalityByNumber(ctx, number)
}
