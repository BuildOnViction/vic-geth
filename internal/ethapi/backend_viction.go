package ethapi

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/rpc"
)

// BackendViction extends Backend with Viction/PoSV helpers for eth RPC.
// Full-node *eth.EthAPIBackend implements both Backend and BackendViction.
// When the engine is *posv.Posv and ChainConfig.Posv is set, GetAPIs registers
// PublicVictionBlockChainAPI on the "eth" namespace.
type BackendViction interface {
	Backend
	GetRewardByHash(ctx context.Context, hash common.Hash) (*posv.EpochReward, error)
	GetAttestorsPairsByHash(ctx context.Context, hash common.Hash) (map[common.Address]common.Address, error)
	GetAttestorsPairsByNumber(ctx context.Context, number rpc.BlockNumber) (map[common.Address]common.Address, error)
	GetBlockFinalityByHash(ctx context.Context, blockHash common.Hash) (uint, error)
	GetBlockFinalityByNumber(ctx context.Context, blockNumber rpc.BlockNumber) (uint, error)
	AreTwoBlockSamePath(hash1, hash2 common.Hash) bool
}
