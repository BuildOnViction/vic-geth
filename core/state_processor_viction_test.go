// Copyright 2015 The go-ethereum Authors
// (original work)
// Copyright 2025 The Viction Authors
// (modifications)
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/legacy/trading/tradingstate"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"
)

// --- Helpers ---

// testChainConfig returns a minimal ChainConfig for production-level tests.
func testChainConfig() *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:               big.NewInt(88),
		TIPNativeTradingBlock: big.NewInt(20_581_700),
		AtlasBlock:            big.NewInt(97_705_094),
		Posv:                  &params.PosvConfig{Period: 2, Epoch: 900, Gap: 5},
		Viction: &params.VictionConfig{
			TradingStateContract: common.HexToAddress("0x0000000000000000000000000000000000000092"),
		},
	}
}

// mockTradingEngine satisfies TradingEngine with no-op methods.
type mockTradingEngine struct{}

func (m *mockTradingEngine) CommitOrder(
	_ *types.Header, _ common.Address,
	_ tradingstate.ChainContext, _ *state.StateDB,
	_ *tradingstate.TradingStateDB, _ common.Hash, _ *tradingstate.OrderItem,
) ([]map[string]string, []*tradingstate.OrderItem, error) {
	return nil, nil, nil
}
func (m *mockTradingEngine) GetTradingState(_ *types.Block, _ common.Address) (*tradingstate.TradingStateDB, error) {
	return nil, nil
}
func (m *mockTradingEngine) UpdateMediumPriceBeforeEpoch(_ uint64, _ *tradingstate.TradingStateDB, _ *state.StateDB) error {
	return nil
}
func (m *mockTradingEngine) GetTradingStateRoot(_ *types.Block, _ common.Address) (common.Hash, error) {
	return tradingstate.EmptyRoot, nil
}
func (m *mockTradingEngine) GetStateCache() tradingstate.Database {
	return tradingstate.NewDatabase(rawdb.NewMemoryDatabase())
}

// newEmptyTradingStateDB creates a fresh empty TradingStateDB backed by an in-memory DB.
func newEmptyTradingStateDB(t *testing.T) *tradingstate.TradingStateDB {
	t.Helper()
	db := tradingstate.NewDatabase(rawdb.NewMemoryDatabase())
	tdb, err := tradingstate.New(tradingstate.EmptyRoot, db)
	require.NoError(t, err)
	return tdb
}

// mockConsensusEngine satisfies consensus.Engine with no-op methods.
// authorAddr is returned by Author(); set it to match the tx signer.
type mockConsensusEngine struct{ authorAddr common.Address }

func (m *mockConsensusEngine) Author(_ *types.Header) (common.Address, error) {
	return m.authorAddr, nil
}
func (m *mockConsensusEngine) VerifyHeader(_ consensus.ChainHeaderReader, _ *types.Header, _ bool) error {
	return nil
}
func (m *mockConsensusEngine) VerifyHeaders(_ consensus.ChainHeaderReader, _ []*types.Header, _ []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error)
	return abort, results
}
func (m *mockConsensusEngine) VerifyUncles(_ consensus.ChainReader, _ *types.Block) error { return nil }
func (m *mockConsensusEngine) VerifySeal(_ consensus.ChainHeaderReader, _ *types.Header) error {
	return nil
}
func (m *mockConsensusEngine) Prepare(_ consensus.ChainHeaderReader, _ *types.Header) error {
	return nil
}
func (m *mockConsensusEngine) Finalize(_ consensus.ChainHeaderReader, _ *types.Header, _ *state.StateDB, _ []*types.Transaction, _ []*types.Header) {
}
func (m *mockConsensusEngine) FinalizeAndAssemble(_ consensus.ChainHeaderReader, _ *types.Header, _ *state.StateDB, _ []*types.Transaction, _ []*types.Header, _ []*types.Receipt) (*types.Block, error) {
	return nil, nil
}
func (m *mockConsensusEngine) Seal(_ consensus.ChainHeaderReader, _ *types.Block, _ chan<- *types.Block, _ <-chan struct{}) error {
	return nil
}
func (m *mockConsensusEngine) SealHash(_ *types.Header) common.Hash { return common.Hash{} }
func (m *mockConsensusEngine) CalcDifficulty(_ consensus.ChainHeaderReader, _ uint64, _ *types.Header) *big.Int {
	return big.NewInt(0)
}
func (m *mockConsensusEngine) APIs(_ consensus.ChainHeaderReader) []rpc.API { return nil }
func (m *mockConsensusEngine) Close() error                                 { return nil }

// TestAfterProcessRootMismatch verifies that afterProcess returns a non-nil error
// when the computed trading root does not match the 0x92 tx root.
//
// Setup: the 0x92 tx is signed by a known key (so GetTradingStateRoot can
// recover the sender and return fakeRoot). The TradingStateDB is empty
// (IntermediateRoot = EmptyRoot). Since fakeRoot ≠ EmptyRoot, afterProcess
// must return a "trading state root mismatch" error.
func TestAfterProcessRootMismatch(t *testing.T) {
	cfg := testChainConfig()
	tradingAddr := cfg.Viction.TradingStateContract

	// A fake root that differs from EmptyRoot (what an empty TradingStateDB produces)
	fakeRoot := common.HexToHash("0xdeadbeef00000000000000000000000000000000000000000000000000000000")
	require.NotEqual(t, tradingstate.EmptyRoot, fakeRoot)

	// Generate a key to act as block author, sign the 0x92 tx with HomesteadSigner
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	authorAddr := crypto.PubkeyToAddress(key.PublicKey)

	data := make([]byte, 64)
	copy(data[:32], fakeRoot.Bytes())

	// Build a block with a signed 0x92 tx so GetTradingStateRoot recovers authorAddr
	rawTx := types.NewTransaction(0, tradingAddr, big.NewInt(0), 0, big.NewInt(0), data)
	signedTx, err := types.SignTx(rawTx, types.HomesteadSigner{}, key)
	require.NoError(t, err)
	header := &types.Header{Number: big.NewInt(100)}
	block := types.NewBlock(header, []*types.Transaction{signedTx}, nil, nil, new(trie.Trie))

	vp := &VictionProcessor{
		config:         cfg,
		tradingEngine:  &mockTradingEngine{},
		engine:         &mockConsensusEngine{authorAddr: authorAddr},
		tradingStateDB: newEmptyTradingStateDB(t), // IntermediateRoot = EmptyRoot ≠ fakeRoot
	}

	afterErr := vp.AfterProcess(block, nil)
	require.Error(t, afterErr, "AfterProcess must return non-nil error on root mismatch")
	require.Contains(t, afterErr.Error(), "trading state root mismatch")
}

// --- Tests: GetTradingStateRoot ---

// TestApplyTomoXTxMalformedBatch verifies the dispatch-layer pre-screening:
// a 0x91 tx with non-decodable data must NOT be intercepted by ApplyNativeTransaction
// (it should return handled=false so the EVM handles it), and applyTomoXTx itself
// should tolerate an unexpected decode failure gracefully (empty receipt, no error).
func TestApplyTomoXTxMalformedBatch(t *testing.T) {
	cfg := testChainConfig()
	tomoXAddr := common.HexToAddress("0x0000000000000000000000000000000000000091")
	cfg.Viction.TradingContract = tomoXAddr
	header := &types.Header{
		Number:   big.NewInt(20_582_000),
		Coinbase: common.Address{},
	}
	tx := types.NewTransaction(0, tomoXAddr, big.NewInt(0), 0, big.NewInt(0), []byte("NOT_JSON"))
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)

	tdb := newEmptyTradingStateDB(t)
	vp := &VictionProcessor{
		config:         cfg,
		engine:         &mockConsensusEngine{},
		tradingEngine:  &mockTradingEngine{},
		tradingStateDB: tdb,
	}
	rootBefore := tdb.IntermediateRoot()

	// ApplyVictionTransaction must return handled=false for a non-JSON 0x91 tx
	// so it falls through to the normal EVM path.
	var usedGas uint64
	handled, _, _, err, _ := vp.ApplyNativeTransaction(statedb, tx, header, &usedGas)
	require.False(t, handled, "non-JSON 0x91 tx must not be intercepted by viction dispatch")
	require.NoError(t, err)

	// applyTomoXTx itself (called directly) should produce an empty receipt on
	// unexpected decode failure — no error, no state mutation.
	handled, receipt, _, err, _ := vp.applyTradingTransaction(statedb, tx, header, &usedGas, tradingstate.TxMatchBatch{})
	require.True(t, handled)
	require.NoError(t, err, "applyTomoXTx fallthrough must not return an error")
	require.NotNil(t, receipt)
	require.Equal(t, rootBefore, tdb.IntermediateRoot(), "no trading-state mutation on decode failure")
}

// TestBeforeProcessPreTIPNativeTradingNilDB verifies that a block before TIPNativeTrading
// activation does not initialize tradingStateDB (no error, no panic).
func TestBeforeProcessPreTIPNativeTradingNilDB(t *testing.T) {
	cfg := testChainConfig() // TIPNativeTrading at 20_581_700

	header := &types.Header{Number: big.NewInt(100)} // pre-activation
	block := types.NewBlock(header, nil, nil, nil, new(trie.Trie))
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)

	vp := &VictionProcessor{
		config:        cfg,
		tradingEngine: &mockTradingEngine{},
	}

	err := vp.BeforeBlockProcess(block, statedb)
	require.NoError(t, err)
	require.Nil(t, vp.tradingStateDB, "tradingStateDB must be nil before TIPNativeTrading activation")
}
