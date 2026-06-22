package tradingstate

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"
)

func GetLocMappingAtKey(key common.Hash, slot uint64) *big.Int {
	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
	retByte := crypto.Keccak256(key.Bytes(), slotHash.Bytes())
	ret := new(big.Int)
	ret.SetBytes(retByte)
	return ret
}

func GetExRelayerFee(relayerSMC common.Address, relayer common.Address, statedb *state.StateDB) *big.Int {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_fee"])
	locHash := common.BigToHash(locBig)
	return statedb.GetState(relayerSMC, locHash).Big()
}

func GetRelayerOwner(relayerSMC common.Address, relayer common.Address, statedb *state.StateDB) common.Address {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	log.Debug("[TradingState] GetRelayerOwner", "relayer", relayer.Hex(), "slot", slot, "locBig", locBig)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_owner"])
	locHash := common.BigToHash(locBig)
	return common.BytesToAddress(statedb.GetState(relayerSMC, locHash).Bytes())
}

// return true if relayer request to resign and have not withdraw locked fund
func IsResignedRelayer(relayerSMC common.Address, relayer common.Address, statedb *state.StateDB) bool {
	slot := RelayerMappingSlot["RESIGN_REQUESTS"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locHash := common.BigToHash(locBig)
	if statedb.GetState(relayerSMC, locHash) != (common.Hash{}) {
		return true
	}
	return false
}

func GetBaseTokenLength(relayerSMC common.Address, relayer common.Address, statedb *state.StateDB) uint64 {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_fromTokens"])
	locHash := common.BigToHash(locBig)
	return statedb.GetState(relayerSMC, locHash).Big().Uint64()
}

func GetBaseTokenAtIndex(relayerSMC common.Address, relayer common.Address, statedb *state.StateDB, index uint64) common.Address {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_fromTokens"])
	locHash := common.BigToHash(locBig)
	loc := state.StorageLocationOfDynamicArrayElement(state.StorageLocationFromHash(locHash), index, 1)
	return common.BytesToAddress(statedb.GetState(relayerSMC, loc.Hash()).Bytes())
}

func GetQuoteTokenLength(relayerSMC common.Address, relayer common.Address, statedb *state.StateDB) uint64 {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_toTokens"])
	locHash := common.BigToHash(locBig)
	return statedb.GetState(relayerSMC, locHash).Big().Uint64()
}

func GetQuoteTokenAtIndex(relayerSMC common.Address, relayer common.Address, statedb *state.StateDB, index uint64) common.Address {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBig = new(big.Int).Add(locBig, RelayerStructMappingSlot["_toTokens"])
	locHash := common.BigToHash(locBig)
	loc := state.StorageLocationOfDynamicArrayElement(state.StorageLocationFromHash(locHash), index, 1)
	return common.BytesToAddress(statedb.GetState(relayerSMC, loc.Hash()).Bytes())
}

func GetRelayerCount(relayerSMC common.Address, statedb *state.StateDB) uint64 {
	slot := RelayerMappingSlot["RelayerCount"]
	slotHash := common.BigToHash(new(big.Int).SetUint64(slot))
	valueHash := statedb.GetState(relayerSMC, slotHash)
	return new(big.Int).SetBytes(valueHash.Bytes()).Uint64()
}

func GetAllCoinbases(relayerSMC common.Address, statedb *state.StateDB) []common.Address {
	relayerCount := GetRelayerCount(relayerSMC, statedb)
	slot := RelayerMappingSlot["RELAYER_COINBASES"]
	coinbases := []common.Address{}
	for i := uint64(0); i < relayerCount; i++ {
		valueHash := statedb.GetState(relayerSMC, common.BytesToHash(GetLocMappingAtKey(common.BigToHash(big.NewInt(int64(i))), slot).Bytes()))
		coinbases = append(coinbases, common.BytesToAddress(valueHash.Bytes()))
	}
	return coinbases
}
func GetAllTradingPairs(relayerSMC common.Address, statedb *state.StateDB) (map[common.Hash]bool, error) {
	coinbases := GetAllCoinbases(relayerSMC, statedb)
	slot := RelayerMappingSlot["RELAYER_LIST"]
	allPairs := map[common.Hash]bool{}
	for _, coinbase := range coinbases {
		locBig := GetLocMappingAtKey(coinbase.Hash(), slot)
		fromTokenSlot := new(big.Int).Add(locBig, RelayerStructMappingSlot["_fromTokens"])
		fromTokenLength := statedb.GetState(relayerSMC, common.BigToHash(fromTokenSlot)).Big().Uint64()
		toTokenSlot := new(big.Int).Add(locBig, RelayerStructMappingSlot["_toTokens"])
		toTokenLength := statedb.GetState(relayerSMC, common.BigToHash(toTokenSlot)).Big().Uint64()
		if toTokenLength != fromTokenLength {
			return map[common.Hash]bool{}, fmt.Errorf("[TradingState] Invalid length from token & to toke : from :%d , to :%d ", fromTokenLength, toTokenLength)
		}
		fromTokens := []common.Address{}
		fromTokenSlotHash := common.BytesToHash(fromTokenSlot.Bytes())
		for i := uint64(0); i < fromTokenLength; i++ {
			fromToken := common.BytesToAddress(statedb.GetState(relayerSMC, state.StorageLocationOfDynamicArrayElement(state.StorageLocationFromHash(fromTokenSlotHash), i, uint64(1)).Hash()).Bytes())
			fromTokens = append(fromTokens, fromToken)
		}
		toTokenSlotHash := common.BytesToHash(toTokenSlot.Bytes())
		for i := uint64(0); i < toTokenLength; i++ {
			toToken := common.BytesToAddress(statedb.GetState(relayerSMC, state.StorageLocationOfDynamicArrayElement(state.StorageLocationFromHash(toTokenSlotHash), i, uint64(1)).Hash()).Bytes())

			log.Debug("[TradingState] GetAllTradingPairs all pair info", "from", fromTokens[i].Hex(), "toToken", toToken.Hex())
			allPairs[GetTradingOrderBookHash(fromTokens[i], toToken)] = true
		}
	}
	log.Debug("[TradingState] GetAllTradingPairs", "coinbase", len(coinbases), "allPairs", len(allPairs))
	return allPairs, nil
}

func SubRelayerFee(relayerSMC common.Address, relayer common.Address, fee *big.Int, statedb *state.StateDB) error {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)

	locBigDeposit := new(big.Int).SetUint64(uint64(0)).Add(locBig, RelayerStructMappingSlot["_deposit"])
	locHashDeposit := common.BigToHash(locBigDeposit)
	balance := statedb.GetState(relayerSMC, locHashDeposit).Big()
	log.Debug("[TradingState] ApplyTomoXMatchedTransaction settle balance: SubRelayerFee BEFORE", "relayer", relayer.String(), "balance", balance)
	if balance.Cmp(fee) < 0 {
		return errors.Errorf("[TradingState] relayer %s isn't enough tomo fee", relayer.String())
	} else {
		balance = new(big.Int).Sub(balance, fee)
		statedb.SetState(relayerSMC, locHashDeposit, common.BigToHash(balance))
		statedb.SubBalance(relayerSMC, fee)
		log.Debug("[TradingState] ApplyTomoXMatchedTransaction settle balance: SubRelayerFee AFTER", "relayer", relayer.String(), "balance", balance)
		return nil
	}
}

func CheckRelayerFee(relayerSMC common.Address, relayer common.Address, fee *big.Int, statedb *state.StateDB) error {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)

	locBigDeposit := new(big.Int).SetUint64(uint64(0)).Add(locBig, RelayerStructMappingSlot["_deposit"])
	locHashDeposit := common.BigToHash(locBigDeposit)
	balance := statedb.GetState(relayerSMC, locHashDeposit).Big()
	if new(big.Int).Sub(balance, fee).Cmp(new(big.Int).Mul(BasePrice, RelayerLockedFund)) < 0 {
		return errors.Errorf("[TradingState] relayer %s isn't enough tomo fee : balance %d , fee : %d ", relayer.Hex(), balance.Uint64(), fee.Uint64())
	}
	return nil
}
func AddTokenBalance(addr common.Address, value *big.Int, token common.Address, statedb *state.StateDB) error {
	// TOMO native
	if token.String() == TomoNativeAddress {
		balance := statedb.GetBalance(addr)
		log.Debug("[TradingState] ApplyTomoXMatchedTransaction settle balance: ADD TOKEN TOMO NATIVE BEFORE", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		statedb.AddBalance(addr, value)
		balance = statedb.GetBalance(addr)
		log.Debug("[TradingState] ApplyTomoXMatchedTransaction settle balance: ADD TOMO NATIVE BALANCE AFTER", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)

		return nil
	}

	// TRC tokens
	if statedb.Exist(token) {
		slot := TokenMappingSlot["balances"]
		locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
		balance := statedb.GetState(token, locHash).Big()
		log.Debug("[TradingState] ApplyTomoXMatchedTransaction settle balance: ADD TOKEN BALANCE BEFORE", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		balance = new(big.Int).Add(balance, value)
		statedb.SetState(token, locHash, common.BigToHash(balance))
		log.Debug("[TradingState] ApplyTomoXMatchedTransaction settle balance: ADD TOKEN BALANCE AFTER", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		return nil
	} else {
		return errors.Errorf("[TradingState] token %s isn't exist", token.String())
	}
}

func SubTokenBalance(addr common.Address, value *big.Int, token common.Address, statedb *state.StateDB) error {
	// TOMO native
	if token.String() == TomoNativeAddress {

		balance := statedb.GetBalance(addr)
		log.Debug("[TradingState] ApplyTomoXMatchedTransaction settle balance: SUB TOMO NATIVE BALANCE BEFORE", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		if balance.Cmp(value) < 0 {
			return errors.Errorf("[TradingState] value %s in token %s not enough , have : %s , want : %s  ", addr.String(), token.String(), balance, value)
		}
		statedb.SubBalance(addr, value)
		balance = statedb.GetBalance(addr)
		log.Debug("[TradingState] ApplyTomoXMatchedTransaction settle balance: SUB TOMO NATIVE BALANCE AFTER", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		return nil
	}

	// TRC tokens
	if statedb.Exist(token) {
		slot := TokenMappingSlot["balances"]
		locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
		balance := statedb.GetState(token, locHash).Big()
		log.Debug("[TradingState] ApplyTomoXMatchedTransaction settle balance: SUB TOKEN BALANCE BEFORE", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		if balance.Cmp(value) < 0 {
			return errors.Errorf("[TradingState] value %s in token %s not enough , have : %s , want : %s  ", addr.String(), token.String(), balance, value)
		}
		balance = new(big.Int).Sub(balance, value)
		statedb.SetState(token, locHash, common.BigToHash(balance))
		log.Debug("[TradingState] ApplyTomoXMatchedTransaction settle balance: SUB TOKEN BALANCE AFTER", "token", token.String(), "address", addr.String(), "balance", balance, "orderValue", value)
		return nil
	} else {
		return errors.Errorf("[TradingState] token %s isn't exist", token.String())
	}
}

func CheckSubTokenBalance(addr common.Address, value *big.Int, token common.Address, statedb *state.StateDB, mapBalances map[common.Address]map[common.Address]*big.Int) (*big.Int, error) {
	// TOMO native
	if token.String() == TomoNativeAddress {
		var balance *big.Int
		if value := mapBalances[token][addr]; value != nil {
			balance = value
		} else {
			balance = statedb.GetBalance(addr)
		}
		if balance.Cmp(value) < 0 {
			return nil, errors.Errorf("[TradingState] value %s in token %s not enough , have : %s , want : %s  ", addr.String(), token.String(), balance, value)
		}
		newBalance := new(big.Int).Sub(balance, value)
		log.Debug("[TradingState] CheckSubTokenBalance settle balance: SUB TOMO NATIVE BALANCE ", "token", token.String(), "address", addr.String(), "balance", balance, "value", value, "newBalance", newBalance)
		return newBalance, nil
	}
	// TRC tokens
	if statedb.Exist(token) {
		var balance *big.Int
		if value := mapBalances[token][addr]; value != nil {
			balance = value
		} else {
			slot := TokenMappingSlot["balances"]
			locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
			balance = statedb.GetState(token, locHash).Big()
		}
		if balance.Cmp(value) < 0 {
			return nil, errors.Errorf("[TradingState] value %s in token %s not enough , have : %s , want : %s  ", addr.String(), token.String(), balance, value)
		}
		newBalance := new(big.Int).Sub(balance, value)
		log.Debug("[TradingState] CheckSubTokenBalance settle balance: SUB TOKEN BALANCE ", "token", token.String(), "address", addr.String(), "balance", balance, "value", value, "newBalance", newBalance)
		return newBalance, nil
	} else {
		return nil, errors.Errorf("[TradingState] token %s isn't exist", token.String())
	}
}

func CheckAddTokenBalance(addr common.Address, value *big.Int, token common.Address, statedb *state.StateDB, mapBalances map[common.Address]map[common.Address]*big.Int) (*big.Int, error) {
	// TOMO native
	if token.String() == TomoNativeAddress {
		var balance *big.Int
		if value := mapBalances[token][addr]; value != nil {
			balance = value
		} else {
			balance = statedb.GetBalance(addr)
		}
		newBalance := new(big.Int).Add(balance, value)
		log.Debug("[TradingState] CheckAddTokenBalance settle balance: ADD TOMO NATIVE BALANCE ", "token", token.String(), "address", addr.String(), "balance", balance, "value", value, "newBalance", newBalance)
		return newBalance, nil
	}
	// TRC tokens
	if statedb.Exist(token) {
		var balance *big.Int
		if value := mapBalances[token][addr]; value != nil {
			balance = value
		} else {
			slot := TokenMappingSlot["balances"]
			locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
			balance = statedb.GetState(token, locHash).Big()
		}
		newBalance := new(big.Int).Add(balance, value)
		log.Debug("[TradingState] CheckAddTokenBalance settle balance: ADD TOKEN BALANCE ", "token", token.String(), "address", addr.String(), "balance", balance, "value", value, "newBalance", newBalance)
		if common.BigToHash(newBalance).Big().Cmp(newBalance) != 0 {
			return nil, fmt.Errorf("[TradingState] Overflow when try add token balance , max is 2^256 , balance : %v , value:%v ", balance, value)
		} else {
			return newBalance, nil
		}
	} else {
		return nil, errors.Errorf("[TradingState] token %s isn't exist", token.String())
	}
}

func CheckSubRelayerFee(relayerSMC common.Address, relayer common.Address, fee *big.Int, statedb *state.StateDB, mapBalances map[common.Address]*big.Int) (*big.Int, error) {
	balance := mapBalances[relayer]
	if balance == nil {
		slot := RelayerMappingSlot["RELAYER_LIST"]
		locBig := GetLocMappingAtKey(relayer.Hash(), slot)
		locBigDeposit := new(big.Int).SetUint64(uint64(0)).Add(locBig, RelayerStructMappingSlot["_deposit"])
		locHashDeposit := common.BigToHash(locBigDeposit)
		balance = statedb.GetState(relayerSMC, locHashDeposit).Big()
	}
	log.Debug("[TradingState] CheckSubRelayerFee settle balance: SubRelayerFee ", "relayer", relayer.String(), "balance", balance, "fee", fee)
	if balance.Cmp(fee) < 0 {
		return nil, errors.Errorf("[TradingState] relayer %s isn't enough tomo fee", relayer.String())
	} else {
		return new(big.Int).Sub(balance, fee), nil
	}
}

func GetTokenBalance(addr common.Address, token common.Address, statedb *state.StateDB) *big.Int {
	// TOMO native
	if token.String() == TomoNativeAddress {
		return statedb.GetBalance(addr)
	}
	// TRC tokens
	if statedb.Exist(token) {
		slot := TokenMappingSlot["balances"]
		locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
		return statedb.GetState(token, locHash).Big()
	} else {
		return common.Big0
	}
}

func SetTokenBalance(addr common.Address, balance *big.Int, token common.Address, statedb *state.StateDB) error {
	// TOMO native
	if token.String() == TomoNativeAddress {
		statedb.SetBalance(addr, balance)
		return nil
	}

	// TRC tokens
	if statedb.Exist(token) {
		slot := TokenMappingSlot["balances"]
		locHash := common.BigToHash(GetLocMappingAtKey(addr.Hash(), slot))
		statedb.SetState(token, locHash, common.BigToHash(balance))
		return nil
	} else {
		return errors.Errorf("[TradingState] token %s isn't exist", token.String())
	}
}

func SetSubRelayerFee(relayerSMC common.Address, relayer common.Address, balance *big.Int, fee *big.Int, statedb *state.StateDB) {
	slot := RelayerMappingSlot["RELAYER_LIST"]
	locBig := GetLocMappingAtKey(relayer.Hash(), slot)
	locBigDeposit := new(big.Int).SetUint64(uint64(0)).Add(locBig, RelayerStructMappingSlot["_deposit"])
	locHashDeposit := common.BigToHash(locBigDeposit)
	statedb.SetState(relayerSMC, locHashDeposit, common.BigToHash(balance))
	statedb.SubBalance(relayerSMC, fee)
}
