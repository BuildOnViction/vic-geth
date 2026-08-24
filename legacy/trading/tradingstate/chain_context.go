package tradingstate

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// ChainContext provides the subset of *Blockchain state for using in Trading/Lending.
type ChainContext interface {
	// CurrentHeader retrieves the current head header of the canonical chain.
	CurrentHeader() *types.Header

	// Config returns the chain configuration, needed for hardfork block checks
	// and Viction-specific contract addresses.
	Config() *params.ChainConfig
}
