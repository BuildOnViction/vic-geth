package lendingstate

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/legacy/trading/tradingstate"
	"github.com/ethereum/go-ethereum/log"
)

var (
	LendingRelayerListSlot    = uint64(0)
	CollateralMapSlot         = uint64(1)
	DefaultCollateralSlot     = uint64(2)
	SupportedBaseSlot         = uint64(3)
	SupportedTermSlot         = uint64(4)
	ILOCollateralSlot         = uint64(5)
	LendingRelayerStructSlots = map[string]*big.Int{
		"fee":         big.NewInt(0),
		"bases":       big.NewInt(1),
		"terms":       big.NewInt(2),
		"collaterals": big.NewInt(3),
	}
	CollateralStructSlots = map[string]*big.Int{
		"depositRate":     big.NewInt(0),
		"liquidationRate": big.NewInt(1),
		"recallRate":      big.NewInt(2),
		"price":           big.NewInt(3),
	}
	PriceStructSlots = map[string]*big.Int{
		"price":       big.NewInt(0),
		"blockNumber": big.NewInt(1),
	}
)

// locSimpleVariable returns the storage location of a simple (non-mapping, non-array) variable at the given slot.
func locSimpleVariable(slot uint64) common.Hash {
	return common.BigToHash(new(big.Int).SetUint64(slot))
}

// locDynamicArrAtElement returns the storage location of an element in a dynamic array.
func locDynamicArrAtElement(slotHash common.Hash, index uint64, elementSize uint64) common.Hash {
	slotKecBig := crypto.Keccak256Hash(slotHash.Bytes()).Big()
	arrBig := slotKecBig.Add(slotKecBig, new(big.Int).SetUint64(index*elementSize))
	return common.BigToHash(arrBig)
}

// locOfStructElement returns the storage location of a struct field.
func locOfStructElement(locOfStruct *big.Int, elementSlotInStruct *big.Int) common.Hash {
	return common.BigToHash(new(big.Int).Add(locOfStruct, elementSlotInStruct))
}

// @function IsValidRelayer : return whether the given address is the coinbase of a valid relayer or not
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @return: true if it's a valid coinbase address of lending protocol, otherwise return false
func IsValidRelayer(lendingSMC common.Address, relayerSMC common.Address, statedb *state.StateDB, coinbase common.Address) bool {
	locRelayerState := GetLocMappingAtKey(coinbase.Hash(), LendingRelayerListSlot)

	// a valid relayer must have baseToken
	locBaseToken := locOfStructElement(locRelayerState, LendingRelayerStructSlots["bases"])
	if v := statedb.GetState(lendingSMC, common.BytesToHash(locBaseToken.Bytes())); v != (common.Hash{}) {
		if tradingstate.IsResignedRelayer(relayerSMC, coinbase, statedb) {
			return false
		}
		slot := tradingstate.RelayerMappingSlot["RELAYER_LIST"]
		locRelayerStateTrading := GetLocMappingAtKey(coinbase.Hash(), slot)

		locBigDeposit := new(big.Int).SetUint64(uint64(0)).Add(locRelayerStateTrading, tradingstate.RelayerStructMappingSlot["_deposit"])
		locHashDeposit := common.BigToHash(locBigDeposit)
		balance := statedb.GetState(relayerSMC, locHashDeposit).Big()
		expectedFund := new(big.Int).Mul(tradingstate.BasePrice, tradingstate.RelayerLockedFund)
		if balance.Cmp(expectedFund) <= 0 {
			log.Debug("Relayer is not in relayer list", "relayer", coinbase.String(), "balance", balance, "expected", expectedFund)
			return false
		}
		return true
	}
	return false
}

// @function GetFee
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @return: feeRate of lending
func GetFee(lendingSMC common.Address, statedb *state.StateDB, coinbase common.Address) *big.Int {
	locRelayerState := GetLocMappingAtKey(coinbase.Hash(), LendingRelayerListSlot)
	locHash := common.BytesToHash(new(big.Int).Add(locRelayerState, LendingRelayerStructSlots["fee"]).Bytes())
	return statedb.GetState(lendingSMC, locHash).Big()
}

// @function GetBaseList
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @return: list of base tokens
func GetBaseList(lendingSMC common.Address, statedb *state.StateDB, coinbase common.Address) []common.Address {
	baseList := []common.Address{}
	locRelayerState := GetLocMappingAtKey(coinbase.Hash(), LendingRelayerListSlot)
	locBaseHash := locOfStructElement(locRelayerState, LendingRelayerStructSlots["bases"])
	length := statedb.GetState(lendingSMC, locBaseHash).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := locDynamicArrAtElement(locBaseHash, i, 1)
		addr := common.BytesToAddress(statedb.GetState(lendingSMC, loc).Bytes())
		if addr != (common.Address{}) {
			baseList = append(baseList, addr)
		}
	}
	return baseList
}

// @function GetTerms
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @return: list of supported terms of the given relayer
func GetTerms(lendingSMC common.Address, statedb *state.StateDB, coinbase common.Address) []uint64 {
	terms := []uint64{}
	locRelayerState := GetLocMappingAtKey(coinbase.Hash(), LendingRelayerListSlot)
	locTermHash := locOfStructElement(locRelayerState, LendingRelayerStructSlots["terms"])
	length := statedb.GetState(lendingSMC, locTermHash).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := locDynamicArrAtElement(locTermHash, i, 1)
		t := statedb.GetState(lendingSMC, loc).Big().Uint64()
		if t != uint64(0) {
			terms = append(terms, t)
		}
	}
	return terms
}

// @function IsValidPair
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @param baseToken: address of baseToken
// @param terms: term
// @return: TRUE if the given baseToken, term organize a valid pair
func IsValidPair(lendingSMC common.Address, statedb *state.StateDB, coinbase common.Address, baseToken common.Address, term uint64) (valid bool, pairIndex uint64) {
	baseTokenList := GetBaseList(lendingSMC, statedb, coinbase)
	terms := GetTerms(lendingSMC, statedb, coinbase)
	baseIndexes := []uint64{}
	for i := uint64(0); i < uint64(len(baseTokenList)); i++ {
		if baseTokenList[i] == baseToken {
			baseIndexes = append(baseIndexes, i)
		}
	}
	for _, index := range baseIndexes {
		if terms[index] == term {
			pairIndex = index
			return true, pairIndex
		}
	}
	return false, pairIndex
}

// @function GetCollaterals
// @param statedb : current state
// @param coinbase: coinbase address of relayer
// @param baseToken: address of baseToken
// @param terms: term
// @return:
//   - collaterals []common.Address  : list of addresses of collateral
//   - isSpecialCollateral           : TRUE if collateral is a token which is NOT available for trading in TomoX, otherwise FALSE
func GetCollaterals(lendingSMC common.Address, statedb *state.StateDB, coinbase common.Address, baseToken common.Address, term uint64) (collaterals []common.Address, isSpecialCollateral bool) {
	validPair, _ := IsValidPair(lendingSMC, statedb, coinbase, baseToken, term)
	if !validPair {
		return []common.Address{}, false
	}

	// if collaterals is not defined for the relayer, return default collaterals
	locDefaultCollateralHash := locSimpleVariable(DefaultCollateralSlot)
	length := statedb.GetState(lendingSMC, locDefaultCollateralHash).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := locDynamicArrAtElement(locDefaultCollateralHash, i, 1)
		addr := common.BytesToAddress(statedb.GetState(lendingSMC, loc).Bytes())
		if addr != (common.Address{}) {
			collaterals = append(collaterals, addr)
		}
	}
	return collaterals, false
}

// @function GetCollateralDetail
// @param statedb : current state
// @param token: address of collateral token
// @return: depositRate, liquidationRate, price of collateral
func GetCollateralDetail(lendingSMC common.Address, statedb *state.StateDB, token common.Address) (depositRate, liquidationRate, recallRate *big.Int) {
	collateralState := GetLocMappingAtKey(token.Hash(), CollateralMapSlot)
	locDepositRate := locOfStructElement(collateralState, CollateralStructSlots["depositRate"])
	locLiquidationRate := locOfStructElement(collateralState, CollateralStructSlots["liquidationRate"])
	locRecallRate := locOfStructElement(collateralState, CollateralStructSlots["recallRate"])
	depositRate = statedb.GetState(lendingSMC, locDepositRate).Big()
	liquidationRate = statedb.GetState(lendingSMC, locLiquidationRate).Big()
	recallRate = statedb.GetState(lendingSMC, locRecallRate).Big()
	return depositRate, liquidationRate, recallRate
}

func GetCollateralPrice(lendingSMC common.Address, statedb *state.StateDB, collateralToken common.Address, lendingToken common.Address) (price, blockNumber *big.Int) {
	collateralState := GetLocMappingAtKey(collateralToken.Hash(), CollateralMapSlot)
	locMapPrices := collateralState.Add(collateralState, CollateralStructSlots["price"])
	locLendingTokenPriceByte := crypto.Keccak256(lendingToken.Hash().Bytes(), common.BigToHash(locMapPrices).Bytes())

	locCollateralPrice := common.BigToHash(new(big.Int).Add(new(big.Int).SetBytes(locLendingTokenPriceByte), PriceStructSlots["price"]))
	locBlockNumber := common.BigToHash(new(big.Int).Add(new(big.Int).SetBytes(locLendingTokenPriceByte), PriceStructSlots["blockNumber"]))

	price = statedb.GetState(lendingSMC, locCollateralPrice).Big()
	blockNumber = statedb.GetState(lendingSMC, locBlockNumber).Big()
	return price, blockNumber
}

// @function GetSupportedTerms
// @param statedb : current state
// @return: list of terms which tomoxlending supports
func GetSupportedTerms(lendingSMC common.Address, statedb *state.StateDB) []uint64 {
	terms := []uint64{}
	locSupportedTerm := locSimpleVariable(SupportedTermSlot)
	length := statedb.GetState(lendingSMC, locSupportedTerm).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := locDynamicArrAtElement(locSupportedTerm, i, 1)
		t := statedb.GetState(lendingSMC, loc).Big().Uint64()
		if t != 0 {
			terms = append(terms, t)
		}
	}
	return terms
}

// @function GetSupportedBaseToken
// @param statedb : current state
// @return: list of tokens which are available for lending
func GetSupportedBaseToken(lendingSMC common.Address, statedb *state.StateDB) []common.Address {
	baseTokens := []common.Address{}
	locSupportedBaseToken := locSimpleVariable(SupportedBaseSlot)
	length := statedb.GetState(lendingSMC, locSupportedBaseToken).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := locDynamicArrAtElement(locSupportedBaseToken, i, 1)
		addr := common.BytesToAddress(statedb.GetState(lendingSMC, loc).Bytes())
		if addr != (common.Address{}) {
			baseTokens = append(baseTokens, addr)
		}
	}
	return baseTokens
}

// @function GetAllCollateral
// @param statedb : current state
// @return: list of address of collateral token
func GetAllCollateral(lendingSMC common.Address, statedb *state.StateDB) []common.Address {
	collaterals := []common.Address{}

	locDefaultCollateralHash := locSimpleVariable(DefaultCollateralSlot)
	length := statedb.GetState(lendingSMC, locDefaultCollateralHash).Big().Uint64()
	for i := uint64(0); i < length; i++ {
		loc := locDynamicArrAtElement(locDefaultCollateralHash, i, 1)
		addr := common.BytesToAddress(statedb.GetState(lendingSMC, loc).Bytes())
		if addr != (common.Address{}) {
			collaterals = append(collaterals, addr)
		}
	}
	return collaterals
}

// @function GetAllLendingBooks
// @param statedb : current state
// @return: a map to specify whether lendingBook (combination of baseToken and term) is valid or not
func GetAllLendingBooks(lendingSMC common.Address, statedb *state.StateDB) (mapLendingBook map[common.Hash]bool, err error) {
	mapLendingBook = make(map[common.Hash]bool)
	baseTokens := GetSupportedBaseToken(lendingSMC, statedb)
	terms := GetSupportedTerms(lendingSMC, statedb)
	if len(baseTokens) == 0 {
		return nil, fmt.Errorf("GetAllLendingBooks: empty baseToken list")
	}
	if len(terms) == 0 {
		return nil, fmt.Errorf("GetAllLendingPairs: empty term list")
	}
	for _, baseToken := range baseTokens {
		for _, term := range terms {
			if (baseToken != common.Address{}) && (term > 0) {
				mapLendingBook[GetLendingOrderBookHash(baseToken, term)] = true
			}
		}
	}
	return mapLendingBook, nil
}

// @function GetAllLendingPairs
// @param statedb : current state
// @return: list of lendingPair (combination of baseToken and collateralToken)
func GetAllLendingPairs(lendingSMC common.Address, statedb *state.StateDB) (allPairs []LendingPair, err error) {
	baseTokens := GetSupportedBaseToken(lendingSMC, statedb)
	collaterals := GetAllCollateral(lendingSMC, statedb)
	if len(baseTokens) == 0 {
		return allPairs, fmt.Errorf("GetAllLendingPairs: empty baseToken list")
	}
	if len(collaterals) == 0 {
		return allPairs, fmt.Errorf("GetAllLendingPairs: empty collateral list")
	}
	for _, baseToken := range baseTokens {
		for _, collateral := range collaterals {
			if (baseToken != common.Address{}) && (collateral != common.Address{}) {
				allPairs = append(allPairs, LendingPair{
					LendingToken:    baseToken,
					CollateralToken: collateral,
				})
			}
		}
	}
	return allPairs, nil
}
