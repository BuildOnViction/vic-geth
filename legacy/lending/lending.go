package lending

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/legacy/lending/lendingstate"
	"github.com/ethereum/go-ethereum/legacy/trading"
	"github.com/ethereum/go-ethereum/legacy/trading/tradingstate"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	ProtocolName       = "tomoxlending"
	ProtocolVersion    = uint64(1)
	ProtocolVersionStr = "1.0"
)

var (
	ErrNonceTooHigh = errors.New("nonce too high")
	ErrNonceTooLow  = errors.New("nonce too low")
)

type Lending struct {
	db                ethdb.Database
	lendingStateCache lendingstate.Database
	trieCacheLimit    int
	triegc            *prque.Prque
	trading           *trading.Trading

	// config is needed to derive the correct transaction signer (EIP155 vs Homestead)
	// when extracting the lending state root from the 0x92 system transaction.
	config *params.ChainConfig
}

// New creates a Lending engine.
// db is the ethdb passed directly - legacy/trading.Trading does not expose its DB.
// tomox is required for token decimal lookups and price conversions.
// config is the chain configuration, required for correct EIP155 signer derivation.
func New(db ethdb.Database, tomox *trading.Trading, config *params.ChainConfig) *Lending {
	return &Lending{
		db:                db,
		lendingStateCache: lendingstate.NewDatabase(db),
		trieCacheLimit:    100,
		triegc:            prque.New(nil),
		trading:           tomox,
		config:            config,
	}
}

// Version returns the Lending sub-protocols version number.
func (l *Lending) Version() uint64 {
	return ProtocolVersion
}

func (l *Lending) GetLendingState(block *types.Block, author common.Address) (*lendingstate.LendingStateDB, error) {
	root, err := l.GetLendingStateRoot(block, author)
	if err != nil {
		return nil, err
	}
	if l.lendingStateCache == nil {
		return nil, errors.New("Not initialized tomox")
	}
	state, err := lendingstate.New(root, l.lendingStateCache)
	if err != nil {
		log.Info("Not found lending state when GetLendingState", "block", block.Number(), "lendingRoot", root.Hex())
	}
	return state, err
}

func (l *Lending) GetLendingStateRoot(block *types.Block, author common.Address) (common.Hash, error) {
	// The 0x92 system tx is signed by the block coinbase using EIP155 (ChainId is set on mainnet
	// since block 3). Using HomesteadSigner would fail to recover the sender for any post-EIP155 tx.
	signer := types.MakeSigner(l.config, block.Number())
	for _, tx := range block.Transactions() {
		if tx.To() == nil || tx.To().Hex() != tradingstate.TradingStateContract {
			continue
		}
		from, err := types.Sender(signer, tx)
		if err != nil || from != author {
			continue
		}
		if len(tx.Data()) >= 64 {
			return common.BytesToHash(tx.Data()[32:]), nil
		}
	}
	return lendingstate.EmptyRoot, nil
}

func (l *Lending) HasLendingState(block *types.Block, author common.Address) bool {
	root, err := l.GetLendingStateRoot(block, author)
	if err != nil {
		return false
	}
	_, err = l.lendingStateCache.OpenTrie(root)
	if err != nil {
		return false
	}
	return true
}

func (l *Lending) GetStateCache() lendingstate.Database {
	return l.lendingStateCache
}

func (l *Lending) GetTriegc() *prque.Prque {
	return l.triegc
}

func (l *Lending) ProcessLiquidationData(header *types.Header, chain tradingstate.ChainContext, statedb *state.StateDB, tradingState *tradingstate.TradingStateDB, lendingState *lendingstate.LendingStateDB) (updatedTrades map[common.Hash]*lendingstate.LendingTrade, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades []*lendingstate.LendingTrade, err error) {
	blockTime := new(big.Int).SetUint64(header.Time)
	updatedTrades = map[common.Hash]*lendingstate.LendingTrade{} // sum of liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades
	liquidatedTrades = []*lendingstate.LendingTrade{}
	autoRepayTrades = []*lendingstate.LendingTrade{}
	autoTopUpTrades = []*lendingstate.LendingTrade{}
	autoRecallTrades = []*lendingstate.LendingTrade{}

	allPairs, err := lendingstate.GetAllLendingPairs(l.config.Viction.LendingRegistrationContract, statedb)
	if err != nil {
		log.Debug("Not found all trading pairs", "error", err)
		return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, nil
	}
	allLendingBooks, err := lendingstate.GetAllLendingBooks(l.config.Viction.LendingRegistrationContract, statedb)
	if err != nil {
		log.Debug("Not found all lending books", "error", err)
		return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, nil
	}

	// liquidate trades by time
	for lendingBook := range allLendingBooks {
		lowestTime, tradingIds := lendingState.GetLowestLiquidationTime(lendingBook, blockTime)
		log.Debug("ProcessLiquidationData time", "tradeIds", len(tradingIds))
		for lowestTime.Sign() > 0 && lowestTime.Cmp(blockTime) < 0 {
			for _, tradingId := range tradingIds {
				log.Debug("ProcessRepay", "lowestTime", lowestTime, "time", blockTime, "lendingBook", lendingBook.Hex(), "tradingId", tradingId.Hex())
				trade, err := l.ProcessRepayLendingTrade(header, chain, lendingState, statedb, tradingState, lendingBook, tradingId.Big().Uint64())
				if err != nil {
					log.Error("Fail when process payment ", "time", blockTime, "lendingBook", lendingBook.Hex(), "tradingId", tradingId, "error", err)
					return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, err
				}
				if trade != nil && trade.Hash != (common.Hash{}) {
					updatedTrades[trade.Hash] = trade
					switch trade.Status {
					case lendingstate.TradeStatusLiquidated:
						liquidatedTrades = append(liquidatedTrades, trade)
					case lendingstate.TradeStatusClosed:
						autoRepayTrades = append(autoRepayTrades, trade)
					}
				}
			}
			lowestTime, tradingIds = lendingState.GetLowestLiquidationTime(lendingBook, blockTime)
		}
	}

	for _, lendingPair := range allPairs {
		orderbook := tradingstate.GetTradingOrderBookHash(lendingPair.CollateralToken, lendingPair.LendingToken)
		_, collateralPrice, err := l.GetCollateralPrices(header, chain, statedb, tradingState, lendingPair.CollateralToken, lendingPair.LendingToken)
		if err != nil || collateralPrice == nil || collateralPrice.Sign() == 0 {
			log.Error("Fail when get price collateral/lending ", "CollateralToken", lendingPair.CollateralToken.Hex(), "LendingToken", lendingPair.LendingToken.Hex(), "error", err)
			// ignore this pair, do not throw error
			continue
		}
		// liquidate trades
		highestLiquidatePrice, liquidationData := tradingState.GetHighestLiquidationPriceData(orderbook, collateralPrice)
		for highestLiquidatePrice.Sign() > 0 && collateralPrice.Cmp(highestLiquidatePrice) < 0 {
			for lendingBook, tradingIds := range liquidationData {
				for _, tradingIdHash := range tradingIds {
					trade := lendingState.GetLendingTrade(lendingBook, tradingIdHash)
					if trade.AutoTopUp {
						if newTrade, err := l.AutoTopUp(statedb, tradingState, lendingState, lendingBook, tradingIdHash, collateralPrice); err == nil {
							// if this action complete successfully, do not liquidate this trade in this epoch
							log.Debug("AutoTopUp", "borrower", trade.Borrower.Hex(), "collateral", newTrade.CollateralToken.Hex(), "tradingIdHash", tradingIdHash.Hex(), "newLockedAmount", newTrade.CollateralLockedAmount)
							autoTopUpTrades = append(autoTopUpTrades, newTrade)
							updatedTrades[newTrade.Hash] = newTrade
							continue
						}
					}
					log.Debug("LiquidationTrade", "highestLiquidatePrice", highestLiquidatePrice, "lendingBook", lendingBook.Hex(), "tradingIdHash", tradingIdHash.Hex())
					newTrade, err := l.LiquidationTrade(lendingState, statedb, tradingState, lendingBook, tradingIdHash.Big().Uint64())
					if err != nil {
						log.Error("Fail when remove liquidation newTrade", "time", blockTime, "lendingBook", lendingBook.Hex(), "tradingIdHash", tradingIdHash.Hex(), "error", err)
						return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, err
					}
					if newTrade != nil && newTrade.Hash != (common.Hash{}) {
						newTrade.Status = lendingstate.TradeStatusLiquidated
						liquidationData := lendingstate.LiquidationData{
							RecallAmount:      common.Big0,
							LiquidationAmount: newTrade.CollateralLockedAmount,
							CollateralPrice:   collateralPrice,
							Reason:            lendingstate.LiquidatedByPrice,
						}
						extraData, _ := json.Marshal(liquidationData)
						newTrade.ExtraData = string(extraData)
						liquidatedTrades = append(liquidatedTrades, newTrade)
						updatedTrades[newTrade.Hash] = newTrade
					}
				}
			}
			highestLiquidatePrice, liquidationData = tradingState.GetHighestLiquidationPriceData(orderbook, collateralPrice)
		}
		// recall trades
		depositRate, liquidationRate, recallRate := lendingstate.GetCollateralDetail(l.config.Viction.LendingRegistrationContract, statedb, lendingPair.CollateralToken)
		recalLiquidatePrice := new(big.Int).Mul(collateralPrice, lendingstate.BaseRecall)
		recalLiquidatePrice = new(big.Int).Div(recalLiquidatePrice, recallRate)
		newLiquidatePrice := new(big.Int).Mul(collateralPrice, liquidationRate)
		newLiquidatePrice = new(big.Int).Div(newLiquidatePrice, depositRate)
		allLowertLiquidationData := tradingState.GetAllLowerLiquidationPriceData(orderbook, recalLiquidatePrice)
		log.Debug("ProcessLiquidationData", "orderbook", orderbook.Hex(), "collateralPrice", collateralPrice, "recallRate", recallRate, "recalLiquidatePrice", recalLiquidatePrice, "newLiquidatePrice", newLiquidatePrice, "allLowertLiquidationData", len(allLowertLiquidationData))
		for price, liquidationData := range allLowertLiquidationData {
			if price.Sign() > 0 && recalLiquidatePrice.Cmp(price) > 0 {
				for lendingBook, tradingIds := range liquidationData {
					for _, tradingIdHash := range tradingIds {
						log.Debug("Process Recall", "price", price, "lendingBook", lendingBook, "tradingIdHash", tradingIdHash.Hex())
						trade := lendingState.GetLendingTrade(lendingBook, tradingIdHash)
						log.Debug("TestRecall", "borrower", trade.Borrower.Hex(), "lendingToken", trade.LendingToken.Hex(), "collateral", trade.CollateralToken.Hex(), "price", price, "tradingIdHash", tradingIdHash.Hex())
						if trade.AutoTopUp {
							err, _, newTrade := l.ProcessRecallLendingTrade(lendingState, statedb, tradingState, lendingBook, tradingIdHash, newLiquidatePrice)
							if err != nil {
								log.Error("ProcessRecallLendingTrade", "lendingBook", lendingBook.Hex(), "tradingIdHash", tradingIdHash.Hex(), "newLiquidatePrice", newLiquidatePrice, "err", err)
								return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, err
							}
							// if this action complete successfully, do not liquidate this trade in this epoch
							log.Debug("AutoRecall", "borrower", trade.Borrower.Hex(), "collateral", newTrade.CollateralToken.Hex(), "lendingBook", lendingBook.Hex(), "tradingIdHash", tradingIdHash.Hex(), "newLockedAmount", newTrade.CollateralLockedAmount)
							autoRecallTrades = append(autoRecallTrades, newTrade)
							updatedTrades[newTrade.Hash] = newTrade
						}
					}
				}
			}
		}
	}

	log.Debug("ProcessLiquidationData", "updatedTrades", len(updatedTrades), "liquidated", len(liquidatedTrades), "autoRepay", len(autoRepayTrades), "autoTopUp", len(autoTopUpTrades), "autoRecall", len(autoRecallTrades))
	return updatedTrades, liquidatedTrades, autoRepayTrades, autoTopUpTrades, autoRecallTrades, nil
}
