// Copyright 2026 The Vic-geth Authors
package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// mockMsgForVRC25 implements the core.Message interface for testing
type mockMsgForVRC25 struct {
	from       common.Address
	to         *common.Address
	gas        uint64
	gasPrice   *big.Int
	value      *big.Int
	nonce      uint64
	data       []byte
	checkNonce bool
}

func (m *mockMsgForVRC25) From() common.Address { return m.from }
func (m *mockMsgForVRC25) To() *common.Address  { return m.to }
func (m *mockMsgForVRC25) GasPrice() *big.Int   { return m.gasPrice }
func (m *mockMsgForVRC25) Gas() uint64          { return m.gas }
func (m *mockMsgForVRC25) Value() *big.Int      { return m.value }
func (m *mockMsgForVRC25) Nonce() uint64        { return m.nonce }
func (m *mockMsgForVRC25) CheckNonce() bool     { return m.checkNonce }
func (m *mockMsgForVRC25) Data() []byte         { return m.data }

// setupTestStateTransition creates a StateTransition with a mocked environment suitable for VRC25 tests
func setupTestStateTransition(t *testing.T, msg Message) (*StateTransition, *state.StateDB) {
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		t.Fatalf("failed to create statedb: %v", err)
	}

	// Create a chain config with a defined VRC25 contract and GasPrice
	vrc25GasPrice := big.NewInt(100)
	chainConfig := &params.ChainConfig{
		ChainID:       big.NewInt(88),
		VRC25Contract: common.HexToAddress("0x8c0faeb5C6bEd2129b8674F262Fd45c4e9468bee"),
		VRC25GasPrice: vrc25GasPrice,
	}

	// Setup BlockContext and EVM
	header := &vm.BlockContext{
		BlockNumber: big.NewInt(100),
		Difficulty:  big.NewInt(1000000),
		GasLimit:    10000000,
	}
	vmConfig := vm.Config{}
	evm := vm.NewEVM(*header, vm.TxContext{}, statedb, chainConfig, vmConfig)

	gp := new(GasPool)
	gp.AddGas(10000000)

	st := NewStateTransition(evm, msg, gp)
	return st, statedb
}

func TestVRC25_BuyGas_Sponsored(t *testing.T) {
	sponsorAddr := common.HexToAddress("0x8c0faeb5C6bEd2129b8674F262Fd45c4e9468bee")
	targetContract := common.HexToAddress("0xTargetContract")
	userAddr := common.HexToAddress("0xUser")

	gasLimit := uint64(10000)
	gasPrice := big.NewInt(200) // Transaction gas price (native)
	// VRC25 Gas Price is set to 100 in setup
	vrc25GasPrice := big.NewInt(100)

	// Cost = 10000 * 100 = 1,000,000
	initialSponsorBalance := big.NewInt(2000000)

	msg := &mockMsgForVRC25{
		from:     userAddr,
		to:       &targetContract,
		gas:      gasLimit,
		gasPrice: gasPrice,
		value:    big.NewInt(0),
	}

	st, statedb := setupTestStateTransition(t, msg)

	// Set Sponsor Balance in Storage
	// slotTokensState is not exported, but we know it's "tokensState" in map.
	// In core/vrc25_state_transition.go: var slotTokensState = state.SlotVRC25Contract["tokensState"]
	// We need to access this slot. Since it's a package level var in core, we can access it if we are in core package.
	// But it is unexported variable 'slotTokensState' in the same package?
	// The file `core/vrc25_state_transition.go` has `var slotTokensState = state.SlotVRC25Contract["tokensState"]`.
	// Since we are in package `core` (same package), we can access `slotTokensState`.

	feeCapKey := state.GetStorageKeyForMapping(targetContract.Hash(), slotTokensState)
	statedb.SetState(sponsorAddr, feeCapKey, common.BigToHash(initialSponsorBalance))

	// Execute buyVRC25Gas
	if err := st.vrc25BuyGas(); err != nil {
		t.Fatalf("vrc25BuyGas failed: %v", err)
	}

	// 1. Verify Payer is the Sponsor Contract
	if st.payer != sponsorAddr {
		t.Errorf("expected payer to be sponsor %s, got %s", sponsorAddr.Hex(), st.payer.Hex())
	}

	// 2. Verify GasPrice is updated to VRC25GasPrice
	if st.gasPrice.Cmp(vrc25GasPrice) != 0 {
		t.Errorf("expected gasPrice to be %v, got %v", vrc25GasPrice, st.gasPrice)
	}

	// 3. Verify Balance Deducted from Storage
	// Expected cost = 10000 * 100 = 1,000,000
	expectedBalance := new(big.Int).Sub(initialSponsorBalance, big.NewInt(1000000))
	currentHash := statedb.GetState(sponsorAddr, feeCapKey)
	currentBalance := currentHash.Big()

	if currentBalance.Cmp(expectedBalance) != 0 {
		t.Errorf("expected storage balance %v, got %v", expectedBalance, currentBalance)
	}
}

func TestVRC25_BuyGas_InsufficientSponsorBalance(t *testing.T) {
	sponsorAddr := common.HexToAddress("0x8c0faeb5C6bEd2129b8674F262Fd45c4e9468bee")
	targetContract := common.HexToAddress("0xTargetContract")
	userAddr := common.HexToAddress("0xUser")

	gasLimit := uint64(10000)
	// Cost = 1,000,000
	initialSponsorBalance := big.NewInt(500000) // Insufficient

	msg := &mockMsgForVRC25{
		from:     userAddr,
		to:       &targetContract,
		gas:      gasLimit,
		gasPrice: big.NewInt(200),
		value:    big.NewInt(0),
	}

	st, statedb := setupTestStateTransition(t, msg)

	feeCapKey := state.GetStorageKeyForMapping(targetContract.Hash(), slotTokensState)
	statedb.SetState(sponsorAddr, feeCapKey, common.BigToHash(initialSponsorBalance))

	if err := st.vrc25BuyGas(); err != nil {
		t.Fatalf("vrc25BuyGas failed: %v", err)
	}

	// Verify Fallback to User
	if st.payer != userAddr {
		t.Errorf("expected payer to be user %s, got %s", userAddr.Hex(), st.payer.Hex())
	}

	// Verify Balance Unchanged
	currentBalance := statedb.GetState(sponsorAddr, feeCapKey).Big()
	if currentBalance.Cmp(initialSponsorBalance) != 0 {
		t.Errorf("expected balance to remain %v, got %v", initialSponsorBalance, currentBalance)
	}
}

func TestVRC25_RefundGas(t *testing.T) {
	sponsorAddr := common.HexToAddress("0x8c0faeb5C6bEd2129b8674F262Fd45c4e9468bee")
	targetContract := common.HexToAddress("0xTargetContract")
	userAddr := common.HexToAddress("0xUser")

	gasLimit := uint64(10000)
	vrc25GasPrice := big.NewInt(100)

	// Assume buyGas happened and deducted full amount (1,000,000)
	// Remaining balance in storage = 1,000,000
	storageBalance := big.NewInt(1000000)

	msg := &mockMsgForVRC25{
		from:     userAddr,
		to:       &targetContract,
		gas:      gasLimit,
		gasPrice: big.NewInt(200), // Original tx gas price (ignored for refund calculation in VRC25?)
		value:    big.NewInt(0),
	}

	st, statedb := setupTestStateTransition(t, msg)

	// Simulate state after execution
	st.gasPrice = vrc25GasPrice // Set to VRC25 price as done in buyGas
	st.payer = sponsorAddr
	st.gas = 4000 // 4000 gas remaining (refund)

	feeCapKey := state.GetStorageKeyForMapping(targetContract.Hash(), slotTokensState)
	statedb.SetState(sponsorAddr, feeCapKey, common.BigToHash(storageBalance))

	remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)
	st.vrc25RefundGas(remaining)

	// Verify Refund
	// Refund Value = 4000 * 100 = 400,000
	expectedBalance := new(big.Int).Add(storageBalance, big.NewInt(400000))

	currentBalance := statedb.GetState(sponsorAddr, feeCapKey).Big()
	if currentBalance.Cmp(expectedBalance) != 0 {
		t.Errorf("expected refund balance %v, got %v", expectedBalance, currentBalance)
	}
}
