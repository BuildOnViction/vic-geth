package ethapi

import (
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
	GetRewardByHash(hash common.Hash) (*posv.EpochReward, error)
	GetAttestorsPairsByHash(hash common.Hash) (map[common.Address]common.Address, error)
	GetAttestorsPairsByNumber(number rpc.BlockNumber) (map[common.Address]common.Address, error)
	GetAttestorsByHashAtCheckPoint(hash common.Hash) ([]int64, error)
	GetAttestorsByNumberAtCheckPoint(number rpc.BlockNumber) ([]int64, error)
	GetPenaltiesByHashAtCheckPoint(hash common.Hash) ([]common.Address, error)
	GetPenaltiesByNumberAtCheckPoint(number rpc.BlockNumber) ([]common.Address, error)
}
