package tomox

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/legacy/tomox/tradingstate"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"sync"

	"github.com/ethereum/go-ethereum/common"
	lru "github.com/hashicorp/golang-lru"
)

const (
	ProtocolName       = "tomox"
	ProtocolVersion    = uint64(1)
	ProtocolVersionStr = "1.0"
	overflowIdx        // Indicator of message queue overflow
	defaultCacheLimit  = 1024
	MaximumTxMatchSize = 1000
)

var (
	ErrNonceTooHigh = errors.New("nonce too high")
	ErrNonceTooLow  = errors.New("nonce too low")
)

type Config struct {
	DataDir string `toml:",omitempty"`
}

// DefaultConfig represents (shocker!) the default configuration.
var DefaultConfig = Config{
	DataDir: "",
}

type TomoX struct {
	// Order related
	db         TomoXDAO
	Triegc     *prque.Prque          // Priority queue mapping block numbers to tries to gc
	StateCache tradingstate.Database // State database to reuse between imports (contains state cache)    *tomox_state.TradingStateDB

	// config is needed to derive the correct transaction signer (EIP155 vs Homestead)
	// when extracting the trading state root from the 0x92 system transaction.
	config *params.ChainConfig

	orderNonce map[common.Address]*big.Int

	settings          sync.Map // holds configuration settings that can be dynamically changed
	tokenDecimalCache *lru.Cache
	orderCache        *lru.Cache
}

func NewWithDB(db ethdb.Database, config *params.ChainConfig) *TomoX {
	tokenDecimalCache, _ := lru.New(defaultCacheLimit)
	orderCache, _ := lru.New(tradingstate.OrderCacheLimit)
	tomoX := &TomoX{
		config:            config,
		orderNonce:        make(map[common.Address]*big.Int),
		Triegc:            prque.New(nil),
		tokenDecimalCache: tokenDecimalCache,
		orderCache:        orderCache,
	}
	tomoX.StateCache = tradingstate.NewDatabase(db)
	tomoX.settings.Store(overflowIdx, false)

	return tomoX
}

func (tomox *TomoX) GetTradingState(block *types.Block, author common.Address) (*tradingstate.TradingStateDB, error) {
	root, err := tomox.GetTradingStateRoot(block, author)
	if err != nil {
		return nil, err
	}
	if tomox.StateCache == nil {
		return nil, errors.New("Not initialized tomox")
	}
	return tradingstate.New(root, tomox.StateCache)
}

func (tomox *TomoX) GetTradingStateRoot(block *types.Block, author common.Address) (common.Hash, error) {
	signer := types.MakeSigner(tomox.config, block.Number())
	for _, tx := range block.Transactions() {
		if tx.To() == nil || tx.To().Hex() != tradingstate.TradingStateAddr {
			continue
		}
		from, err := types.Sender(signer, tx)
		if err != nil || from != author {
			continue
		}
		if len(tx.Data()) >= 32 {
			return common.BytesToHash(tx.Data()[:32]), nil
		}
	}
	return tradingstate.EmptyRoot, nil
}

// return average price of the given pair in the last epoch
func (tomox *TomoX) GetAveragePriceLastEpoch(chain tradingstate.ChainContext, statedb *state.StateDB, tradingStateDb *tradingstate.TradingStateDB, baseToken common.Address, quoteToken common.Address) (*big.Int, error) {
	price := tradingStateDb.GetMediumPriceBeforeEpoch(tradingstate.GetTradingOrderBookHash(baseToken, quoteToken))
	if price != nil && price.Sign() > 0 {
		log.Debug("GetAveragePriceLastEpoch", "baseToken", baseToken.Hex(), "quoteToken", quoteToken.Hex(), "price", price)
		return price, nil
	} else {
		inversePrice := tradingStateDb.GetMediumPriceBeforeEpoch(tradingstate.GetTradingOrderBookHash(quoteToken, baseToken))
		log.Debug("GetAveragePriceLastEpoch", "baseToken", baseToken.Hex(), "quoteToken", quoteToken.Hex(), "inversePrice", inversePrice)
		if inversePrice != nil && inversePrice.Sign() > 0 {
			quoteTokenDecimal, err := tomox.GetTokenDecimal(chain, statedb, quoteToken)
			if err != nil || quoteTokenDecimal.Sign() == 0 {
				return nil, fmt.Errorf("fail to get tokenDecimal. Token: %v . Err: %v", quoteToken.String(), err)
			}
			baseTokenDecimal, err := tomox.GetTokenDecimal(chain, statedb, baseToken)
			if err != nil || baseTokenDecimal.Sign() == 0 {
				return nil, fmt.Errorf("fail to get tokenDecimal. Token: %v . Err: %v", baseToken.String(), err)
			}
			price = new(big.Int).Mul(baseTokenDecimal, quoteTokenDecimal)
			price = new(big.Int).Div(price, inversePrice)
			log.Debug("GetAveragePriceLastEpoch", "baseToken", baseToken.Hex(), "quoteToken", quoteToken.Hex(), "baseTokenDecimal", baseTokenDecimal, "quoteTokenDecimal", quoteTokenDecimal, "inversePrice", inversePrice)
			return price, nil
		}
	}
	return nil, nil
}

// GetStateCache returns the trie-node cache backed by the tomox LevelDB.
func (tomox *TomoX) GetStateCache() tradingstate.Database {
	return tomox.StateCache
}

// GetTriegc returns the garbage-collection priority queue for the trading trie.
func (tomox *TomoX) GetTriegc() *prque.Prque {
	return tomox.Triegc
}

// return tokenQuantity (after convert from TOMO to token), tokenPriceInTOMO, error
func (tomox *TomoX) ConvertTOMOToToken(chain tradingstate.ChainContext, statedb *state.StateDB, tradingStateDb *tradingstate.TradingStateDB, token common.Address, quantity *big.Int) (*big.Int, *big.Int, error) {
	if token.String() == tradingstate.TomoNativeAddress {
		return quantity, tradingstate.BasePrice, nil
	}
	tokenPriceInTomo, err := tomox.GetAveragePriceLastEpoch(chain, statedb, tradingStateDb, token, common.HexToAddress(tradingstate.TomoNativeAddress))
	if err != nil || tokenPriceInTomo == nil || tokenPriceInTomo.Sign() <= 0 {
		return common.Big0, common.Big0, err
	}

	tokenDecimal, err := tomox.GetTokenDecimal(chain, statedb, token)
	if err != nil || tokenDecimal.Sign() == 0 {
		return common.Big0, common.Big0, fmt.Errorf("fail to get tokenDecimal. Token: %v . Err: %v", token.String(), err)
	}
	tokenQuantity := new(big.Int).Mul(quantity, tokenDecimal)
	tokenQuantity = new(big.Int).Div(tokenQuantity, tokenPriceInTomo)
	return tokenQuantity, tokenPriceInTomo, nil
}
