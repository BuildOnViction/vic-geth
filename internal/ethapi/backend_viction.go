package ethapi

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
)

// BackendViction extends Backend with Viction/PoSV helpers for eth RPC.
// Full-node *eth.EthAPIBackend implements both Backend and BackendViction.
// When the engine is *posv.Posv and ChainConfig.Posv is set, GetAPIs registers
// PublicVictionBlockChainAPI on the "eth" namespace.
type BackendViction interface {
	Backend
	GetRewardByHash(hash common.Hash) *posv.EpochReward
	GetVotersRewards(common.Address, common.Hash) map[common.Address]*big.Int
	GetEpochDuration() *big.Int
	GetMasternodesCap(checkpoint uint64) map[common.Address]*big.Int
	GetBlocksHashCache(blockNr uint64) []common.Hash
	AreTwoBlockSamePath(newBlock common.Hash, oldBlock common.Hash) bool
}
