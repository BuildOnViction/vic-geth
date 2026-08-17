package lendingstate

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	tradingstate "github.com/ethereum/go-ethereum/legacy/tomox/tradingstate"
	"github.com/ethereum/go-ethereum/params"
	"golang.org/x/crypto/sha3"
)

const (
	Investing                  = "INVEST"
	Borrowing                  = "BORROW"
	TopUp                      = "TOPUP"
	Repay                      = "REPAY"
	Recall                     = "RECALL"
	LendingStatusNew           = "NEW"
	LendingStatusOpen          = "OPEN"
	LendingStatusReject        = "REJECTED"
	LendingStatusFilled        = "FILLED"
	LendingStatusPartialFilled = "PARTIAL_FILLED"
	LendingStatusCancelled     = "CANCELLED"
	Market                     = "MO"
	Limit                      = "LO"
)

var ValidInputLendingStatus = map[string]bool{
	LendingStatusNew:       true,
	LendingStatusCancelled: true,
}

var ValidInputLendingType = map[string]bool{
	Market: true,
	Limit:  true,
	Repay:  true,
	TopUp:  true,
	Recall: true,
}

// Signature struct
type Signature struct {
	V byte        `json:"v"`
	R common.Hash `json:"r"`
	S common.Hash `json:"s"`
}

type LendingItem struct {
	Quantity        *big.Int       `json:"quantity"`
	Interest        *big.Int       `json:"interest"`
	Side            string         `json:"side"` // INVESTING/BORROWING
	Type            string         `json:"type"` // LIMIT/MARKET
	LendingToken    common.Address `json:"lendingToken"`
	CollateralToken common.Address `json:"collateralToken"`
	AutoTopUp       bool           `json:"autoTopUp"`
	FilledAmount    *big.Int       `json:"filledAmount"`
	Status          string         `json:"status"`
	Relayer         common.Address `json:"relayer"`
	Term            uint64         `json:"term"`
	UserAddress     common.Address `json:"userAddress"`
	Signature       *Signature     `json:"signature"`
	Hash            common.Hash    `json:"hash"`
	TxHash          common.Hash    `json:"txHash"`
	Nonce           *big.Int       `json:"nonce"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	LendingId       uint64         `json:"lendingId"`
	LendingTradeId  uint64         `json:"tradeId"`
	ExtraData       string         `json:"extraData"`
}

func (l *LendingItem) VerifyLendingItem(lendingSMC common.Address, relayerSMC common.Address, state *state.StateDB) error {
	if err := l.VerifyLendingStatus(); err != nil {
		return err
	}
	if valid, _ := IsValidPair(lendingSMC, state, l.Relayer, l.LendingToken, l.Term); valid == false {
		return fmt.Errorf("invalid pair . LendToken %s . Term: %v", l.LendingToken.Hex(), l.Term)
	}
	if l.Status == LendingStatusNew {
		if err := l.VerifyLendingType(); err != nil {
			return err
		}
		if l.Type != Repay {
			if err := l.VerifyLendingQuantity(); err != nil {
				return err
			}
		}
		if l.Type == Limit || l.Type == Market {
			if err := l.VerifyLendingSide(); err != nil {
				return err
			}
			if l.Side == Borrowing {
				if err := l.VerifyCollateral(lendingSMC, state); err != nil {
					return err
				}
			}
		}
		if l.Type == Limit {
			if err := l.VerifyLendingInterest(); err != nil {
				return err
			}
		}
	}
	if !IsValidRelayer(lendingSMC, relayerSMC, state, l.Relayer) {
		return fmt.Errorf("VerifyLendingItem: invalid relayer. address: %s", l.Relayer.Hex())
	}
	return nil
}

func (l *LendingItem) VerifyLendingSide() error {
	if l.Side != Borrowing && l.Side != Investing {
		return fmt.Errorf("VerifyLendingSide: invalid side . Side: %s", l.Side)
	}
	return nil
}

func (l *LendingItem) VerifyCollateral(lendingSMC common.Address, state *state.StateDB) error {
	if l.CollateralToken.String() == EmptyAddress || l.CollateralToken.String() == l.LendingToken.String() {
		return fmt.Errorf("invalid collateral %s", l.CollateralToken.Hex())
	}
	validCollateral := false
	collateralList, _ := GetCollaterals(lendingSMC, state, l.Relayer, l.LendingToken, l.Term)
	for _, collateral := range collateralList {
		if l.CollateralToken.String() == collateral.String() {
			validCollateral = true
			break
		}
	}
	if !validCollateral {
		return fmt.Errorf("invalid collateral %s", l.CollateralToken.Hex())
	}
	return nil
}

func (l *LendingItem) VerifyLendingInterest() error {
	if l.Interest == nil || l.Interest.Sign() <= 0 {
		return fmt.Errorf("VerifyLendingInterest: invalid interest. Interest: %v", l.Interest)
	}
	return nil
}

func (l *LendingItem) VerifyLendingQuantity() error {
	if l.Quantity == nil || l.Quantity.Sign() <= 0 {
		return fmt.Errorf("VerifyLendingQuantity: invalid quantity. Quantity: %v", l.Quantity)
	}
	return nil
}

func (l *LendingItem) VerifyLendingType() error {
	if valid, ok := ValidInputLendingType[l.Type]; !ok && !valid {
		return fmt.Errorf("VerifyLendingType: invalid lending type. Type: %s", l.Type)
	}
	return nil
}

func (l *LendingItem) VerifyLendingStatus() error {
	if valid, ok := ValidInputLendingStatus[l.Status]; !ok && !valid {
		return fmt.Errorf("VerifyLendingStatus: invalid lending status. Status: %s", l.Status)
	}
	return nil
}

func (l *LendingItem) ComputeHash() common.Hash {
	sha := sha3.NewLegacyKeccak256().(crypto.KeccakState)
	if l.Status == LendingStatusNew {
		sha.Write(l.Relayer.Bytes())
		sha.Write(l.UserAddress.Bytes())
		sha.Write(l.LendingToken.Bytes())
		sha.Write(l.CollateralToken.Bytes())
		sha.Write([]byte(strconv.FormatInt(int64(l.Term), 10)))
		sha.Write(common.BigToHash(l.Quantity).Bytes())
		if l.Type == Limit {
			if l.Interest != nil {
				sha.Write(common.BigToHash(l.Interest).Bytes())
			}
		}
		sha.Write(common.BigToHash(l.EncodedSide()).Bytes())
		sha.Write([]byte(l.Status))
		sha.Write([]byte(l.Type))
		sha.Write(common.BigToHash(l.Nonce).Bytes())
	} else if l.Status == LendingStatusCancelled {
		sha.Write(l.Hash.Bytes())
		sha.Write(common.BigToHash(l.Nonce).Bytes())
		sha.Write(l.UserAddress.Bytes())
		sha.Write(common.BigToHash(big.NewInt(int64(l.LendingId))).Bytes())
		sha.Write([]byte(l.Status))
		sha.Write(l.Relayer.Bytes())
		sha.Write(l.LendingToken.Bytes())
		sha.Write(l.CollateralToken.Bytes())
	} else {
		return common.Hash{}
	}

	var result [32]byte
	sha.Read(result[:])
	return common.BytesToHash(result[:])
}

func (l *LendingItem) EncodedSide() *big.Int {
	if l.Side == Borrowing {
		return big.NewInt(0)
	}
	return big.NewInt(1)
}

func VerifyBalance(isNativeLendingFork bool, lendingSMC common.Address, statedb *state.StateDB, lendingStateDb *LendingStateDB,
	orderType, side, status string, userAddress, relayer, lendingToken, collateralToken common.Address,
	quantity, lendingTokenDecimal, collateralTokenDecimal, lendTokenTOMOPrice, collateralPrice *big.Int,
	term uint64, lendingId uint64, lendingTradeId uint64) error {
	borrowingFeeRate := GetFee(lendingSMC, statedb, relayer)
	switch orderType {
	case TopUp:
		lendingBook := GetLendingOrderBookHash(lendingToken, term)
		lendingTrade := lendingStateDb.GetLendingTrade(lendingBook, params.Uint64ToHash(lendingTradeId))
		if lendingTrade == EmptyLendingTrade {
			return fmt.Errorf("VerifyBalance: process deposit for emptyLendingTrade is not allowed. lendingTradeId: %v", lendingTradeId)
		}
		tokenBalance := GetTokenBalance(lendingTrade.Borrower, lendingTrade.CollateralToken, statedb)
		if tokenBalance.Cmp(quantity) < 0 {
			return fmt.Errorf("VerifyBalance: not enough balance to process deposit for lendingTrade."+
				"lendingTradeId: %v. Token: %s. ExpectedBalance: %s. ActualBalance: %s",
				lendingTradeId, lendingTrade.CollateralToken.Hex(), quantity.String(), tokenBalance.String())
		}
	case Repay:
		lendingBook := GetLendingOrderBookHash(lendingToken, term)
		lendingTrade := lendingStateDb.GetLendingTrade(lendingBook, params.Uint64ToHash(lendingTradeId))
		if lendingTrade == EmptyLendingTrade {
			return fmt.Errorf("VerifyBalance: process payment for emptyLendingTrade is not allowed. lendingTradeId: %v", lendingTradeId)
		}
		tokenBalance := GetTokenBalance(lendingTrade.Borrower, lendingTrade.LendingToken, statedb)
		paymentBalance := CalculateTotalRepayValue(uint64(time.Now().Unix()), lendingTrade.LiquidationTime, lendingTrade.Term, lendingTrade.Interest, lendingTrade.Amount)

		if tokenBalance.Cmp(paymentBalance) < 0 {
			return fmt.Errorf("VerifyBalance: not enough balance to process payment for lendingTrade."+
				"lendingTradeId: %v. Token: %s. ExpectedBalance: %s. ActualBalance: %s",
				lendingTradeId, lendingTrade.LendingToken.Hex(), paymentBalance.String(), tokenBalance.String())

		}
	case Market, Limit:
		switch side {
		case Investing:
			switch status {
			case LendingStatusNew:
				// make sure that investor have enough lendingToken
				if balance := GetTokenBalance(userAddress, lendingToken, statedb); balance.Cmp(quantity) < 0 {
					return fmt.Errorf("VerifyBalance: investor doesn't have enough lendingToken. User: %s. Token: %s. Expected: %v. Have: %v", userAddress.Hex(), lendingToken.Hex(), quantity, balance)
				}
				// check quantity: reject if it's too small
				if lendTokenTOMOPrice != nil && lendTokenTOMOPrice.Sign() > 0 {
					defaultFee := new(big.Int).Mul(quantity, new(big.Int).SetUint64(DefaultFeeRate))
					defaultFee = new(big.Int).Div(defaultFee, tradingstate.TomoXBaseFee)
					defaultFeeInTOMO := common.Big0
					if lendingToken.String() != tradingstate.TomoNativeAddress {
						defaultFeeInTOMO = new(big.Int).Mul(defaultFee, lendTokenTOMOPrice)
						defaultFeeInTOMO = new(big.Int).Div(defaultFeeInTOMO, lendingTokenDecimal)
					} else {
						defaultFeeInTOMO = defaultFee
					}
					if defaultFeeInTOMO.Cmp(RelayerLendingFee) <= 0 {
						return ErrQuantityTradeTooSmall
					}

				}

			case LendingStatusCancelled:
				// in case of cancel, investor need to pay cancel fee in lendingToken
				// make sure actualBalance >= cancel fee
				lendingBook := GetLendingOrderBookHash(lendingToken, term)
				item := lendingStateDb.GetLendingOrder(lendingBook, common.BigToHash(new(big.Int).SetUint64(lendingId)))
				cancelFee := big.NewInt(0)
				cancelFee = new(big.Int).Mul(item.Quantity, borrowingFeeRate)
				cancelFee = new(big.Int).Div(cancelFee, tradingstate.TomoXBaseCancelFee)

				actualBalance := GetTokenBalance(userAddress, lendingToken, statedb)
				if actualBalance.Cmp(cancelFee) < 0 {
					return fmt.Errorf("VerifyBalance: investor doesn't have enough lendingToken to pay cancel fee. LendingToken: %s . ExpectedBalance: %s . ActualBalance: %s",
						lendingToken.Hex(), cancelFee.String(), actualBalance.String())
				}
			default:
				return fmt.Errorf("VerifyBalance: invalid status of investing lendingitem. Status: %s", status)
			}
			return nil
		case Borrowing:
			switch status {
			case LendingStatusNew:
				depositRate, _, _ := GetCollateralDetail(lendingSMC, statedb, collateralToken)
				settleBalanceResult, err := GetSettleBalance(isNativeLendingFork, Borrowing, lendTokenTOMOPrice, collateralPrice, depositRate, borrowingFeeRate, lendingToken, collateralToken, lendingTokenDecimal, collateralTokenDecimal, quantity)
				if err != nil {
					return err
				}
				expectedBalance := settleBalanceResult.CollateralLockedAmount
				actualBalance := GetTokenBalance(userAddress, collateralToken, statedb)
				if actualBalance.Cmp(expectedBalance) < 0 {
					return fmt.Errorf("VerifyBalance: borrower doesn't have enough collateral token.  User: %s. CollateralToken: %s . ExpectedBalance: %s . ActualBalance: %s",
						userAddress.Hex(), collateralToken.Hex(), expectedBalance.String(), actualBalance.String())
				}
			case LendingStatusCancelled:
				lendingBook := GetLendingOrderBookHash(lendingToken, term)
				item := lendingStateDb.GetLendingOrder(lendingBook, common.BigToHash(new(big.Int).SetUint64(lendingId)))
				cancelFee := big.NewInt(0)
				// Fee ==  quantityToLend/base lend token decimal *price*borrowFee/LendingCancelFee
				cancelFee = new(big.Int).Div(item.Quantity, collateralPrice)
				cancelFee = new(big.Int).Mul(cancelFee, borrowingFeeRate)
				cancelFee = new(big.Int).Div(cancelFee, tradingstate.TomoXBaseCancelFee)
				actualBalance := GetTokenBalance(userAddress, collateralToken, statedb)
				if actualBalance.Cmp(cancelFee) < 0 {
					return fmt.Errorf("VerifyBalance: borrower doesn't have enough collateralToken to pay cancel fee. User: %s. CollateralToken: %s . ExpectedBalance: %s . ActualBalance: %s",
						userAddress.Hex(), lendingToken.Hex(), cancelFee.String(), actualBalance.String())
				}
			default:
				return fmt.Errorf("VerifyBalance: invalid status of borrowing lendingitem. Status: %s", status)
			}
			return nil
		default:
			return fmt.Errorf("VerifyBalance: unknown lending side")
		}
	default:
		return fmt.Errorf("VerifyBalance: unknown lending type")
	}
	return nil
}
