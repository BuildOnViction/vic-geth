package tomox

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/tomox/tradingstate"
)

type TomoX struct{}

func (tomox *TomoX) GetTradingStateRoot(block *types.Block, author common.Address) (common.Hash, error) {
	return common.Hash{}, nil
}

func (tomox *TomoX) GetStateCache() tradingstate.Database {
	return nil
}
