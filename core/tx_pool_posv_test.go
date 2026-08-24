// Copyright 2026 The Vic-geth Authors
// POSV-specific transaction pool tests.
// Tests ensure:
// 1. Old code (non-POSV, Ethash) still works unchanged.
// 2. New code (POSV) handles special transactions (BlockSigner, Randomize) correctly.
// 3. ApplyTransaction routes correctly for both consensus engines.

package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/params"
)

// --- Test chain configs ---

var (
	// blockSignerAddr is the block signer contract address (0x89)
	blockSignerAddr = common.HexToAddress("0x0000000000000000000000000000000000000089")
	// randomizerAddr is the randomize contract address (0x90)
	randomizerAddr = common.HexToAddress("0x0000000000000000000000000000000000000090")

	// testPosvConfig is a minimal POSV chain config for testing.
	testPosvConfig = &params.ChainConfig{
		ChainID:           big.NewInt(69),
		HomesteadBlock:    big.NewInt(0),
		EIP150Block:       big.NewInt(0),
		EIP155Block:       big.NewInt(0),
		EIP158Block:       big.NewInt(0),
		ByzantiumBlock:    big.NewInt(0),
		TIP2019Block:      big.NewInt(0),
		TIPSigningBlock:   big.NewInt(0),
		TIPRandomizeBlock: big.NewInt(0),
		TIPGasPriceBlock:  big.NewInt(0),
		Posv:              &params.PosvConfig{Period: 2, Epoch: 900, Gap: 5},
		Viction: &params.VictionConfig{
			ValidatorBlockSignContract: blockSignerAddr,
			RandomizerContract:         randomizerAddr,
			VRC25GasPrice:              (*math.Decimal256)(big.NewInt(2500)),
		},
	}
)

// setupTxPoolPosv creates a txpool with POSV chain config.
func setupTxPoolPosv() (*TxPool, *ecdsa.PrivateKey) {
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockchain := &testBlockChain{statedb, 84000000, new(event.Feed)}

	key, _ := crypto.GenerateKey()
	pool := NewTxPool(testTxPoolConfig, testPosvConfig, blockchain)
	return pool, key
}

// setupTxPoolEthash creates a txpool with standard Ethash config (non-POSV).
func setupTxPoolEthash() (*TxPool, *ecdsa.PrivateKey) {
	return setupTxPool() // uses params.TestChainConfig (Ethash)
}

// makeBlockSignerTx creates a BlockSigner special tx: sign(uint256 blockNumber, bytes32 blockHash)
func makeBlockSignerTx(nonce uint64, blockNum uint64, blockHash common.Hash, key *ecdsa.PrivateKey) *types.Transaction {
	// ABI: selector(4) + uint256(32) + bytes32(32) = 68 bytes
	data := make([]byte, 68)
	// sign method selector: 0xe341eaa4
	data[0] = 0xe3
	data[1] = 0x41
	data[2] = 0xea
	data[3] = 0xa4
	// uint256 block number (big-endian, last 8 bytes of 32-byte word)
	data[28] = byte(blockNum >> 56)
	data[29] = byte(blockNum >> 48)
	data[30] = byte(blockNum >> 40)
	data[31] = byte(blockNum >> 32)
	data[32] = byte(blockNum >> 24)
	data[33] = byte(blockNum >> 16)
	data[34] = byte(blockNum >> 8)
	data[35] = byte(blockNum)
	// bytes32 block hash
	copy(data[36:68], blockHash.Bytes())

	tx := types.NewTransaction(nonce, blockSignerAddr, big.NewInt(0), 100000, big.NewInt(0), data)
	signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(69)), key)
	return signedTx
}

// makeRandomizeTx creates a Randomize contract tx (e.g. setSecret).
func makeRandomizeTx(nonce uint64, gasPrice *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	// Just a tx to the Randomize contract with some data
	data := make([]byte, 100) // arbitrary payload
	tx := types.NewTransaction(nonce, randomizerAddr, big.NewInt(0), 100000, gasPrice, data)
	signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(69)), key)
	return signedTx
}

// makeRegularTx creates a normal value transfer tx.
func makeRegularTx(nonce uint64, gasPrice *big.Int, key *ecdsa.PrivateKey) *types.Transaction {
	to := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	tx := types.NewTransaction(nonce, to, big.NewInt(1000), 21000, gasPrice, nil)
	signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(69)), key)
	return signedTx
}

// --- Tests: Ethash (non-POSV) still works ---

func TestTxPool_Ethash_RegularTxAccepted(t *testing.T) {
	pool, key := setupTxPoolEthash()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))

	tx := transaction(0, 100000, key)
	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("ethash: regular tx should be accepted, got: %v", err)
	}
	pending, _ := pool.Pending()
	if len(pending[addr]) != 1 {
		// Check if it ended up in the queue (normal for freshly added txs)
		if pool.queue[addr] == nil || pool.queue[addr].Len() != 1 {
			t.Fatal("ethash: regular tx not in pending or queue")
		}
	}
}

func TestTxPool_Ethash_UnderpricedTxRejected(t *testing.T) {
	pool, key := setupTxPoolEthash()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))

	// Set pool min gas price above 0
	pool.SetGasPrice(big.NewInt(100))

	tx := pricedTransaction(0, 100000, big.NewInt(1), key)
	err := pool.AddRemote(tx)
	if err != ErrUnderpriced {
		t.Fatalf("ethash: underpriced tx should be rejected, got: %v", err)
	}
}

func TestTxPool_Ethash_SpecialTxNotExempt(t *testing.T) {
	// On Ethash chains, a "special" tx (to 0x89) is NOT exempt from gas price
	// because Posv is nil → isPosvSpecialTx returns false.
	pool, key := setupTxPoolEthash()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))

	pool.SetGasPrice(big.NewInt(100))

	// Create a tx to blocksigner contract with zero gas price
	data := make([]byte, 68)
	data[0], data[1], data[2], data[3] = 0xe3, 0x41, 0xea, 0xa4
	tx := types.NewTransaction(0, blockSignerAddr, big.NewInt(0), 100000, big.NewInt(0), data)
	signedTx, _ := types.SignTx(tx, types.HomesteadSigner{}, key)

	err := pool.AddRemote(signedTx)
	if err != ErrUnderpriced {
		t.Fatalf("ethash: blocksigner tx should NOT be exempt from gas price, got: %v", err)
	}
}

// --- Tests: POSV special tx handling ---

func TestTxPool_Posv_BlockSignerTxAccepted_ZeroGasPrice(t *testing.T) {
	pool, key := setupTxPoolPosv()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))
	pool.SetGasPrice(big.NewInt(2500))

	// Set IsSigner to recognize this address
	pool.IsSigner = func(a common.Address) bool { return a == addr }

	tx := makeBlockSignerTx(0, 100, common.HexToHash("0xdeadbeef"), key)

	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("posv: block signer tx from signer should be accepted with zero gas price, got: %v", err)
	}
}

func TestTxPool_Posv_BlockSignerTxRejected_NonSigner(t *testing.T) {
	pool, key := setupTxPoolPosv()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))
	pool.SetGasPrice(big.NewInt(2500))

	// This address is NOT a signer
	pool.IsSigner = func(a common.Address) bool { return false }

	tx := makeBlockSignerTx(0, 100, common.HexToHash("0xdeadbeef"), key)

	err := pool.AddRemote(tx)
	if err != ErrUnderpriced {
		t.Fatalf("posv: block signer tx from non-signer should be rejected (underpriced), got: %v", err)
	}
}

func TestTxPool_Posv_RandomizeTxAccepted_ZeroGasPrice(t *testing.T) {
	pool, key := setupTxPoolPosv()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))
	pool.SetGasPrice(big.NewInt(2500))

	pool.IsSigner = func(a common.Address) bool { return a == addr }

	tx := makeRandomizeTx(0, big.NewInt(0), key)

	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("posv: randomize tx from signer should be accepted with zero gas price, got: %v", err)
	}
}

func TestTxPool_Posv_RegularTxRejected_ZeroGasPrice(t *testing.T) {
	pool, key := setupTxPoolPosv()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))
	pool.SetGasPrice(big.NewInt(2500))

	pool.IsSigner = func(a common.Address) bool { return a == addr }

	// Regular tx with zero gas price should still be rejected
	tx := makeRegularTx(0, big.NewInt(0), key)

	err := pool.AddRemote(tx)
	if err != ErrUnderpriced {
		t.Fatalf("posv: regular tx with zero gas price should be rejected, got: %v", err)
	}
}

func TestTxPool_Posv_RegularTxAccepted_SufficientGasPrice(t *testing.T) {
	pool, key := setupTxPoolPosv()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))
	pool.SetGasPrice(big.NewInt(2500))

	tx := makeRegularTx(0, big.NewInt(2500), key)

	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("posv: regular tx with sufficient gas price should be accepted, got: %v", err)
	}
}

func TestTxPool_Posv_SpecialTxSkipsIntrinsicGas(t *testing.T) {
	pool, key := setupTxPoolPosv()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))

	pool.IsSigner = func(a common.Address) bool { return a == addr }

	// Create a blocksigner tx with very low gas limit (below intrinsic gas for 68 bytes of data)
	data := make([]byte, 68)
	data[0], data[1], data[2], data[3] = 0xe3, 0x41, 0xea, 0xa4
	tx := types.NewTransaction(0, blockSignerAddr, big.NewInt(0), 100, big.NewInt(0), data) // gas=100, way below intrinsic
	signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(69)), key)

	// On POSV, special txs skip intrinsic gas check
	if err := pool.AddRemote(signedTx); err != nil {
		t.Fatalf("posv: special tx should skip intrinsic gas check, got: %v", err)
	}
}

func TestTxPool_Posv_SpecialTxPromotedDirectly(t *testing.T) {
	pool, key := setupTxPoolPosv()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))

	pool.IsSigner = func(a common.Address) bool { return a == addr }

	tx := makeBlockSignerTx(0, 100, common.HexToHash("0xdeadbeef"), key)

	if err := pool.AddRemote(tx); err != nil {
		t.Fatalf("posv: special tx should be accepted, got: %v", err)
	}

	// Special txs from signers should be promoted directly to pending
	pending, _ := pool.Pending()
	if len(pending[addr]) == 0 {
		t.Fatal("posv: special tx should be directly in pending")
	}
	if pending[addr][0].Hash() != tx.Hash() {
		t.Fatal("posv: wrong tx in pending")
	}
}

func TestTxPool_Posv_DuplicateSpecialTxRejected(t *testing.T) {
	pool, key := setupTxPoolPosv()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))

	pool.IsSigner = func(a common.Address) bool { return a == addr }

	tx1 := makeBlockSignerTx(0, 100, common.HexToHash("0xdeadbeef"), key)
	tx2 := makeBlockSignerTx(0, 200, common.HexToHash("0xcafebabe"), key) // same nonce, different content

	if err := pool.AddRemote(tx1); err != nil {
		t.Fatalf("posv: first special tx should be accepted, got: %v", err)
	}

	err := pool.AddRemote(tx2)
	if err == nil {
		t.Fatal("posv: duplicate special tx (same nonce) should be rejected")
	}
}

func TestTxPool_Posv_NonceTooLow(t *testing.T) {
	pool, key := setupTxPoolPosv()
	defer pool.Stop()

	addr := crypto.PubkeyToAddress(key.PublicKey)
	pool.currentState.AddBalance(addr, big.NewInt(1000000000))
	pool.currentState.SetNonce(addr, 5) // state nonce = 5

	pool.IsSigner = func(a common.Address) bool { return a == addr }

	// Try to add a tx with nonce 3 (below state nonce)
	tx := makeBlockSignerTx(3, 100, common.HexToHash("0xdeadbeef"), key)

	err := pool.AddRemote(tx)
	if err != ErrNonceTooLow {
		t.Fatalf("posv: tx with nonce too low should be rejected, got: %v", err)
	}
}

// --- Tests: ApplyTransaction ---

func TestApplyTransaction_BlockSignerBypassesEVM(t *testing.T) {
	// Setup a state with an account that has a nonce
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)

	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	statedb.AddBalance(addr, big.NewInt(1000000000))
	statedb.SetNonce(addr, 0)

	header := &types.Header{
		Number:   big.NewInt(1000), // > epoch (900), so signing is active
		GasLimit: 84000000,
	}

	var usedGas uint64
	gp := new(GasPool).AddGas(header.GasLimit)

	tx := makeBlockSignerTx(0, 100, common.HexToHash("0xdeadbeef"), key)
	statedb.Prepare(tx.Hash(), common.Hash{}, 0)

	receipt, err := ApplyTransaction(testPosvConfig, nil, nil, gp, statedb, header, tx, &usedGas, vm.Config{})
	if err != nil {
		t.Fatalf("ApplyTransaction failed: %v", err)
	}
	if receipt == nil {
		t.Fatal("receipt should not be nil")
	}
	// BlockSigner tx should have zero gas used
	if receipt.GasUsed != 0 {
		t.Fatalf("expected gasUsed=0 for blocksigner tx, got: %d", receipt.GasUsed)
	}
	// Total usedGas should not increase
	if usedGas != 0 {
		t.Fatalf("expected total usedGas=0 after blocksigner tx, got: %d", usedGas)
	}
	// Nonce should be incremented
	if statedb.GetNonce(addr) != 1 {
		t.Fatalf("expected nonce=1 after blocksigner tx, got: %d", statedb.GetNonce(addr))
	}
	// Should have a log entry
	if len(receipt.Logs) == 0 {
		t.Fatal("expected at least one log entry in receipt")
	}
	if receipt.Logs[0].Address != blockSignerAddr {
		t.Fatalf("log address should be blocksigner contract, got: %v", receipt.Logs[0].Address)
	}
}

func TestApplyTransaction_RegularTxGoesEVM(t *testing.T) {
	// For a regular tx, ApplyTransaction should route to normal EVM path.
	// We just verify it doesn't bypass (i.e., gas is consumed).
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)

	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	statedb.AddBalance(addr, big.NewInt(10000000000))
	statedb.SetNonce(addr, 0)

	header := &types.Header{
		Number:     big.NewInt(1000),
		GasLimit:   84000000,
		Difficulty: big.NewInt(1),
		Time:       1000,
	}

	var usedGas uint64
	gp := new(GasPool).AddGas(header.GasLimit)

	to := common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678")
	tx := types.NewTransaction(0, to, big.NewInt(100), 21000, big.NewInt(2500), nil)
	signedTx, _ := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(69)), key)
	statedb.Prepare(signedTx.Hash(), common.Hash{}, 0)

	// We need a ChainContext for the EVM path. Use a minimal mock.
	bc := &applyTxTestChain{config: testPosvConfig}
	author := common.HexToAddress("0xaaaa")

	receipt, err := ApplyTransaction(testPosvConfig, bc, &author, gp, statedb, header, signedTx, &usedGas, vm.Config{})
	if err != nil {
		t.Fatalf("ApplyTransaction for regular tx failed: %v", err)
	}
	if receipt == nil {
		t.Fatal("receipt should not be nil")
	}
	// Regular tx should consume gas (at least 21000 for transfer)
	if receipt.GasUsed != 21000 {
		t.Fatalf("expected gasUsed=21000 for transfer, got: %d", receipt.GasUsed)
	}
	if usedGas != 21000 {
		t.Fatalf("expected total usedGas=21000, got: %d", usedGas)
	}
}

func TestApplyTransaction_NonPosvConfig_RegularPath(t *testing.T) {
	// With Ethash config (Posv=nil, Viction=nil), ApplyTransaction should
	// always go through normal EVM path even for txs addressed to 0x89.
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)

	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	statedb.AddBalance(addr, big.NewInt(10000000000))
	statedb.SetNonce(addr, 0)

	header := &types.Header{
		Number:     big.NewInt(1000),
		GasLimit:   10000000,
		Difficulty: big.NewInt(1),
		Time:       1000,
	}

	var usedGas uint64
	gp := new(GasPool).AddGas(header.GasLimit)

	// A tx to blocksigner address but on non-POSV chain
	data := make([]byte, 68)
	data[0], data[1], data[2], data[3] = 0xe3, 0x41, 0xea, 0xa4
	tx := types.NewTransaction(0, blockSignerAddr, big.NewInt(0), 100000, big.NewInt(1), data)
	signedTx, _ := types.SignTx(tx, types.HomesteadSigner{}, key)
	statedb.Prepare(signedTx.Hash(), common.Hash{}, 0)

	bc := &applyTxTestChain{config: params.TestChainConfig}
	author := common.HexToAddress("0xbbbb")

	// On non-POSV, Viction is nil so the bypass is skipped → goes to EVM
	receipt, err := ApplyTransaction(params.TestChainConfig, bc, &author, gp, statedb, header, signedTx, &usedGas, vm.Config{})
	if err != nil {
		t.Fatalf("ApplyTransaction on ethash should work: %v", err)
	}
	// Gas should be consumed (EVM path)
	if receipt.GasUsed == 0 {
		t.Fatal("ethash: tx to 0x89 should go through EVM and consume gas")
	}
}

func TestApplySignTransaction_NonceMismatch(t *testing.T) {
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)

	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	statedb.AddBalance(addr, big.NewInt(1000000000))
	statedb.SetNonce(addr, 5)

	header := &types.Header{
		Number:   big.NewInt(1000),
		GasLimit: 84000000,
	}

	var usedGas uint64

	// Tx with nonce 3 (too low)
	tx := makeBlockSignerTx(3, 100, common.HexToHash("0xdeadbeef"), key)
	statedb.Prepare(tx.Hash(), common.Hash{}, 0)

	_, receipt, _, err, _ := (&VictionProcessor{config: testPosvConfig}).applyBlockSigningTransaction(tx, header, statedb, &usedGas)
	if err == nil {
		t.Fatal("expected error for nonce too low")
	}
	if receipt != nil {
		t.Fatal("receipt should be nil on error")
	}

	// Tx with nonce 10 (too high)
	tx2 := makeBlockSignerTx(10, 100, common.HexToHash("0xdeadbeef"), key)
	statedb.Prepare(tx2.Hash(), common.Hash{}, 0)

	_, receipt2, _, err2, _ := (&VictionProcessor{config: testPosvConfig}).applyBlockSigningTransaction(tx2, header, statedb, &usedGas)
	if err2 == nil {
		t.Fatal("expected error for nonce too high")
	}
	if receipt2 != nil {
		t.Fatal("receipt should be nil on error")
	}

	// Correct nonce = 5
	tx3 := makeBlockSignerTx(5, 100, common.HexToHash("0xdeadbeef"), key)
	statedb.Prepare(tx3.Hash(), common.Hash{}, 0)

	_, receipt3, _, err3, _ := (&VictionProcessor{config: testPosvConfig}).applyBlockSigningTransaction(tx3, header, statedb, &usedGas)
	if err3 != nil {
		t.Fatalf("expected no error for correct nonce, got: %v", err3)
	}
	if receipt3 == nil {
		t.Fatal("receipt should not be nil for correct nonce")
	}
	if statedb.GetNonce(addr) != 6 {
		t.Fatalf("nonce should be incremented to 6, got: %d", statedb.GetNonce(addr))
	}
}

// --- Test helpers ---

// applyTxTestChain is a minimal ChainContext for ApplyTransaction tests.
type applyTxTestChain struct {
	config *params.ChainConfig
}

func (c *applyTxTestChain) Engine() consensus.Engine { return nil }

func (c *applyTxTestChain) GetHeader(common.Hash, uint64) *types.Header {
	return &types.Header{
		Number:   big.NewInt(999),
		GasLimit: 84000000,
	}
}
