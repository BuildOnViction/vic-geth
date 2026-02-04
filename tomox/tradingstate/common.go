package tradingstate

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

const (
	OrderCacheLimit = 10000
)

// tradingExchangeObject is the Ethereum consensus representation of exchanges.
// These objects are stored in the main orderId trie.
type orderList struct {
	Volume *big.Int
	Root   common.Hash // merkle root of the storage trie
}

// tradingExchangeObject is the Ethereum consensus representation of exchanges.
// These objects are stored in the main orderId trie.
type tradingExchangeObject struct {
	Nonce                  uint64
	LastPrice              *big.Int
	MediumPriceBeforeEpoch *big.Int
	MediumPrice            *big.Int
	TotalQuantity          *big.Int
	LendingCount           *big.Int
	AskRoot                common.Hash // merkle root of the storage trie
	BidRoot                common.Hash // merkle root of the storage trie
	OrderRoot              common.Hash
	LiquidationPriceRoot   common.Hash
}
