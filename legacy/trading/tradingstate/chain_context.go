package tradingstate

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// ChainContext provides the subset of blockchain state that the TomoX legacy
// engine needs during order matching. It is intentionally minimal — only the
// methods that legacy/trading actually calls are included.
//
// Both the core-level TradingEngine interface and the legacy/trading engine
// use this type to avoid coupling legacy/trading to the consensus package.
type ChainContext interface {
	// CurrentHeader retrieves the current head header of the canonical chain.
	CurrentHeader() *types.Header

	// Config returns the chain configuration, needed for hardfork block checks
	// and Viction-specific contract addresses.
	Config() *params.ChainConfig
}
