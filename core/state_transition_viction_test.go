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

// mockMsg implements the core.Message interface for testing
type mockMsg struct {
	from       common.Address
	to         *common.Address
	gas        uint64
	gasPrice   *big.Int
	value      *big.Int
	nonce      uint64
	data       []byte
	checkNonce bool
}

func (m *mockMsg) From() common.Address { return m.from }
func (m *mockMsg) To() *common.Address  { return m.to }
func (m *mockMsg) GasPrice() *big.Int   { return m.gasPrice }
func (m *mockMsg) Gas() uint64          { return m.gas }
func (m *mockMsg) Value() *big.Int      { return m.value }
func (m *mockMsg) Nonce() uint64        { return m.nonce }
func (m *mockMsg) CheckNonce() bool     { return m.checkNonce }
func (m *mockMsg) Data() []byte         { return m.data }

// newTestStateTransition creates a StateTransition with a mocked environment suitable for VRC25 tests
func newTestStateTransition(t *testing.T, msg Message) (*StateTransition, *state.StateDB) {
	db := state.NewDatabase(rawdb.NewMemoryDatabase())
	statedb, err := state.New(common.Hash{}, db, nil)
	if err != nil {
		t.Fatalf("failed to create statedb: %v", err)
	}

	// Create a chain config with a defined VRC25 contract
	chainConfig := &params.ChainConfig{
		ChainID:       big.NewInt(88),
		VRC25Contract: common.HexToAddress("0x8c0faeb5C6bEd2129b8674F262Fd45c4e9468bee"),
		VRC25GasPrice: big.NewInt(10),
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

func TestBuyVRC25Gas_HappyPath(t *testing.T) {
	// Setup
	sponsorAddr := common.HexToAddress("0x8c0faeb5C6bEd2129b8674F262Fd45c4e9468bee") // Must match VRC25Contract in config
	targetContract := common.HexToAddress("0xTargetContract")
	userAddr := common.HexToAddress("0xUser")

	gasLimit := uint64(1000)
	gasPrice := big.NewInt(10) // Total Cost = 1000 * 10 = 10000
	initialBalance := big.NewInt(50000)

	msg := &mockMsg{
		from:     userAddr,
		to:       &targetContract,
		gas:      gasLimit,
		gasPrice: gasPrice,
		value:    big.NewInt(0),
	}

	st, statedb := newTestStateTransition(t, msg)

	// 1. Setup VRC25 Storage: Set Fee Capacity for targetContract
	// Slot Key calculation must match core/vrc25_state_transition.go
	feeCapKey := state.GetStorageKeyForMapping(targetContract.Hash(), slotTokensState)
	statedb.SetState(sponsorAddr, feeCapKey, common.BigToHash(initialBalance))

	// 2. Run buyVRC25Gas
	if err := st.vrc25BuyGas(); err != nil {
		t.Fatalf("vrc25BuyGas failed: %v", err)
	}

	// 3. Assertions
	// Check Payer is the Sponsor (System Contract)
	if st.payer != sponsorAddr {
		t.Errorf("expected payer to be sponsor %s, got %s", sponsorAddr.Hex(), st.payer.Hex())
	}

	// Check Storage Balance Deducted
	expectedBalance := new(big.Int).Sub(initialBalance, new(big.Int).Mul(big.NewInt(int64(gasLimit)), gasPrice))
	currentHash := statedb.GetState(sponsorAddr, feeCapKey)
	currentBalance := currentHash.Big()

	if currentBalance.Cmp(expectedBalance) != 0 {
		t.Errorf("expected storage balance %v, got %v", expectedBalance, currentBalance)
	}
}

func TestBuyVRC25Gas_InsufficientFunds(t *testing.T) {
	// Setup
	sponsorAddr := common.HexToAddress("0x8c0faeb5C6bEd2129b8674F262Fd45c4e9468bee")
	targetContract := common.HexToAddress("0xTargetContract")
	userAddr := common.HexToAddress("0xUser")

	gasLimit := uint64(1000)
	gasPrice := big.NewInt(10)         // Cost = 10000
	initialBalance := big.NewInt(5000) // Insufficient Balance

	msg := &mockMsg{
		from:     userAddr,
		to:       &targetContract,
		gas:      gasLimit,
		gasPrice: gasPrice,
		value:    big.NewInt(0),
	}

	st, statedb := newTestStateTransition(t, msg)

	// Set Insufficient Balance
	feeCapKey := state.GetStorageKeyForMapping(targetContract.Hash(), slotTokensState)
	statedb.SetState(sponsorAddr, feeCapKey, common.BigToHash(initialBalance))

	// Run
	if err := st.vrc25BuyGas(); err != nil {
		t.Fatalf("vrc25BuyGas failed: %v", err)
	}

	// Assert: Payer should be USER, not Sponsor
	if st.payer != userAddr {
		t.Errorf("expected fallback to user payer %s, got %s", userAddr.Hex(), st.payer.Hex())
	}

	// Assert: Storage Balance Unchanged
	currentBalance := statedb.GetState(sponsorAddr, feeCapKey).Big()
	if currentBalance.Cmp(initialBalance) != 0 {
		t.Errorf("expected balance to remain %v, got %v", initialBalance, currentBalance)
	}
}

func TestBuyVRC25Gas_NotSponsored(t *testing.T) {
	// Setup
	// sponsorAddr := common.HexToAddress("0x8c0faeb5C6bEd2129b8674F262Fd45c4e9468bee")
	targetContract := common.HexToAddress("0xTargetContract")
	userAddr := common.HexToAddress("0xUser")

	msg := &mockMsg{
		from:     userAddr,
		to:       &targetContract,
		gas:      1000,
		gasPrice: big.NewInt(10),
		value:    big.NewInt(0),
	}

	st, _ := newTestStateTransition(t, msg)

	// NO STORAGE SET for targetContract

	// Run
	if err := st.vrc25BuyGas(); err != nil {
		t.Fatalf("vrc25BuyGas failed: %v", err)
	}

	// Assert: Payer is User
	if st.payer != userAddr {
		t.Errorf("expected payer to be user %s, got %s", userAddr.Hex(), st.payer.Hex())
	}
}

func TestRefundGasSponsoringTransaction(t *testing.T) {
	// Setup
	sponsorAddr := common.HexToAddress("0x8c0faeb5C6bEd2129b8674F262Fd45c4e9468bee")
	targetContract := common.HexToAddress("0xTargetContract")
	userAddr := common.HexToAddress("0xUser")

	gasLimit := uint64(2000)
	gasUsed := uint64(500)
	remainingGas := gasLimit - gasUsed // 1500
	gasPrice := big.NewInt(10)

	// Initial balance AFTER deduction (simulating end of tx execution)
	postDeductionBalance := big.NewInt(80000)

	msg := &mockMsg{
		from:     userAddr,
		to:       &targetContract,
		gas:      gasLimit,
		gasPrice: gasPrice,
		value:    big.NewInt(0),
	}

	st, statedb := newTestStateTransition(t, msg)
	st.gas = remainingGas  // Set remaining gas manually
	st.payer = sponsorAddr // Simulate that it was sponsored

	// Set initial state
	feeCapKey := state.GetStorageKeyForMapping(targetContract.Hash(), slotTokensState)
	statedb.SetState(sponsorAddr, feeCapKey, common.BigToHash(postDeductionBalance))

	// Run Refund
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(remainingGas), gasPrice)
	st.vrc25RefundGas(remaining)

	// Assert
	refundAmount := remaining                                               // 1500 * 10 = 15000
	expectedBalance := new(big.Int).Add(postDeductionBalance, refundAmount) // 80000 + 15000 = 95000

	currentBalance := statedb.GetState(sponsorAddr, feeCapKey).Big()
	if currentBalance.Cmp(expectedBalance) != 0 {
		t.Errorf("expected refund balance %v, got %v", expectedBalance, currentBalance)
	}
}
