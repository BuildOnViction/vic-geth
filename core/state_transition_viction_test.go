// Copyright 2025 The Viction Authors
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
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

var (
	stvVrc25Contract     = common.HexToAddress("0x0000000000000000000000000000000000000099")
	stvValidatorContract = common.HexToAddress("0x0000000000000000000000000000000000000088")
	stvToken             = common.HexToAddress("0x0000000000000000000000000000000000001000")
	stvSender            = common.HexToAddress("0x0000000000000000000000000000000000000200")
	stvCoinbase          = common.HexToAddress("0x0000000000000000000000000000000000000300")
	stvOwner             = common.HexToAddress("0x0000000000000000000000000000000000000400")

	stvTRC21GasPrice    = big.NewInt(2500)
	stvTRC21NewGasPrice = big.NewInt(250000000)
	stvVRC25GasPrice    = big.NewInt(250000000)
)

func vicChainConfig(atlas, postAtlas, tipGasPrice *big.Int) *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:          big.NewInt(88),
		HomesteadBlock:   big.NewInt(0),
		EIP150Block:      big.NewInt(0),
		EIP155Block:      big.NewInt(0),
		EIP158Block:      big.NewInt(0),
		ByzantiumBlock:   big.NewInt(0),
		AtlasBlock:       atlas,
		PostAtlasBlock:   postAtlas,
		TIPGasPriceBlock: tipGasPrice,
		Viction: &params.VictionConfig{
			VRC25Contract:     stvVrc25Contract,
			ValidatorContract: stvValidatorContract,
			TRC21GasPrice:     (*math.Decimal256)(new(big.Int).Set(stvTRC21GasPrice)),
			TRC21NewGasPrice:  (*math.Decimal256)(new(big.Int).Set(stvTRC21NewGasPrice)),
			VRC25GasPrice:     (*math.Decimal256)(new(big.Int).Set(stvVRC25GasPrice)),
		},
	}
}

func newVicST(cfg *params.ChainConfig, blockNum uint64, msg types.Message, feePool map[common.Address]*big.Int) (*StateTransition, *state.StateDB) {
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	blockCtx := vm.BlockContext{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     func(n uint64) common.Hash { return common.Hash{} },
		Coinbase:    stvCoinbase,
		GasLimit:    84000000,
		BlockNumber: new(big.Int).SetUint64(blockNum),
		Time:        big.NewInt(1000),
		Difficulty:  big.NewInt(1),
	}
	txCtx := vm.TxContext{Origin: msg.From(), GasPrice: msg.GasPrice()}
	evm := vm.NewEVM(blockCtx, txCtx, statedb, cfg, vm.Config{})
	st := NewStateTransition(evm, msg, new(GasPool).AddGas(84000000), feePool)
	return st, statedb
}

func newVicMsg(from common.Address, to *common.Address, gas uint64, gasPrice *big.Int) types.Message {
	return types.NewMessage(from, to, 0, big.NewInt(0), gas, gasPrice, nil, false)
}

func TestBuyGasVrc25(t *testing.T) {
	const gas uint64 = 21000
	tokenPtr := &stvToken
	zeroPrice := big.NewInt(0)

	preTIPFee := new(big.Int).Mul(new(big.Int).SetUint64(gas), stvTRC21GasPrice)
	postTIPFee := new(big.Int).Mul(new(big.Int).SetUint64(gas), stvTRC21NewGasPrice)
	vrc25Fee := new(big.Int).Mul(new(big.Int).SetUint64(gas), stvVRC25GasPrice)

	t.Run("nil_viction_config", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		cfg.Viction = nil
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		st, _ := newVicST(cfg, 10, msg, nil)
		require.NoError(t, st.buyGasVrc25())
		require.Equal(t, stvSender, st.payer)
		require.False(t, st.isZeroGasTransaction())
	})

	t.Run("zero_vrc25_contract", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		cfg.Viction.VRC25Contract = common.Address{}
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		st, _ := newVicST(cfg, 10, msg, nil)
		require.NoError(t, st.buyGasVrc25())
		require.False(t, st.isZeroGasTransaction())
	})

	t.Run("pre_atlas_contract_creation", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, nil, gas, zeroPrice)
		st, _ := newVicST(cfg, 10, msg, map[common.Address]*big.Int{stvToken: big.NewInt(1e18)})
		require.NoError(t, st.buyGasVrc25())
		require.False(t, st.isZeroGasTransaction())
	})

	t.Run("pre_atlas_nil_fee_pool", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		st, _ := newVicST(cfg, 10, msg, nil)
		require.NoError(t, st.buyGasVrc25())
		require.False(t, st.isZeroGasTransaction())
	})

	t.Run("pre_atlas_token_absent_from_fee_pool", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		other := common.HexToAddress("0x0000000000000000000000000000000000005000")
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		st, _ := newVicST(cfg, 10, msg, map[common.Address]*big.Int{other: big.NewInt(1e18)})
		require.NoError(t, st.buyGasVrc25())
		require.False(t, st.isZeroGasTransaction())
	})

	t.Run("pre_atlas_nil_fee_cap", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		st, _ := newVicST(cfg, 10, msg, map[common.Address]*big.Int{stvToken: nil})
		require.NoError(t, st.buyGasVrc25())
		require.False(t, st.isZeroGasTransaction())
	})

	t.Run("pre_atlas_pre_tip_cap_below_gasfee", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		feePool := map[common.Address]*big.Int{stvToken: new(big.Int).Sub(preTIPFee, big.NewInt(1))}
		st, _ := newVicST(cfg, 10, msg, feePool)
		require.NoError(t, st.buyGasVrc25())
		require.False(t, st.isZeroGasTransaction())
	})

	t.Run("pre_atlas_pre_tip_cap_equals_gasfee", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		feePool := map[common.Address]*big.Int{stvToken: new(big.Int).Set(preTIPFee)}
		st, _ := newVicST(cfg, 10, msg, feePool)
		require.NoError(t, st.buyGasVrc25())
		require.True(t, st.isZeroGasTransaction())
		require.Equal(t, stvVrc25Contract, st.payer)
		require.Equal(t, 0, stvTRC21GasPrice.Cmp(st.gasPrice))
	})

	t.Run("pre_atlas_post_tip_cap_below_gasfee", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		feePool := map[common.Address]*big.Int{stvToken: new(big.Int).Sub(postTIPFee, big.NewInt(1))}
		st, _ := newVicST(cfg, 60, msg, feePool)
		require.NoError(t, st.buyGasVrc25())
		require.False(t, st.isZeroGasTransaction())
	})

	t.Run("pre_atlas_post_tip_cap_equals_gasfee", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		feePool := map[common.Address]*big.Int{stvToken: new(big.Int).Set(postTIPFee)}
		st, _ := newVicST(cfg, 60, msg, feePool)
		require.NoError(t, st.buyGasVrc25())
		require.True(t, st.isZeroGasTransaction())
		require.Equal(t, stvVrc25Contract, st.payer)
		require.Equal(t, 0, stvTRC21NewGasPrice.Cmp(st.gasPrice))
	})

	t.Run("pre_atlas_tip_boundary_at_fork_block_pre_tip", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		feePool := map[common.Address]*big.Int{stvToken: new(big.Int).Set(preTIPFee)}
		st, _ := newVicST(cfg, 50, msg, feePool)
		require.NoError(t, st.buyGasVrc25())
		require.True(t, st.isZeroGasTransaction())
		require.Equal(t, stvVrc25Contract, st.payer)
		require.Equal(t, 0, stvTRC21GasPrice.Cmp(st.gasPrice))
	})

	t.Run("pre_atlas_tip_boundary_after_fork_block_post_tip", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		feePool := map[common.Address]*big.Int{stvToken: new(big.Int).Set(postTIPFee)}
		st, _ := newVicST(cfg, 51, msg, feePool)
		require.NoError(t, st.buyGasVrc25())
		require.True(t, st.isZeroGasTransaction())
		require.Equal(t, stvVrc25Contract, st.payer)
		require.Equal(t, 0, stvTRC21NewGasPrice.Cmp(st.gasPrice))
	})

	t.Run("post_atlas_zero_capacity", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		st, _ := newVicST(cfg, 150, msg, nil)
		require.NoError(t, st.buyGasVrc25())
		require.False(t, st.isZeroGasTransaction())
	})

	t.Run("post_atlas_cap_equals_fee", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		st, statedb := newVicST(cfg, 150, msg, nil)
		statedb.VicSetZeroGasCapacity(stvVrc25Contract, stvToken, new(big.Int).Set(vrc25Fee))
		require.NoError(t, st.buyGasVrc25())
		require.False(t, st.isZeroGasTransaction())
	})

	t.Run("post_atlas_cap_exceeds_fee", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, zeroPrice)
		st, statedb := newVicST(cfg, 150, msg, nil)
		statedb.VicSetZeroGasCapacity(stvVrc25Contract, stvToken, new(big.Int).Add(vrc25Fee, big.NewInt(1)))
		require.NoError(t, st.buyGasVrc25())
		require.True(t, st.isZeroGasTransaction())
		require.Equal(t, stvVrc25Contract, st.payer)
		require.Equal(t, 0, stvVRC25GasPrice.Cmp(st.gasPrice))
	})
}

func TestRefundGasVrc25(t *testing.T) {
	const (
		gasLimit uint64 = 21000
		gasUsed  uint64 = 12000
	)
	remaining := big.NewInt(1_000_000)
	tokenPtr := &stvToken
	zeroPrice := big.NewInt(0)

	t.Run("pre_atlas_vrc25_noop", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gasLimit, zeroPrice)
		st, statedb := newVicST(cfg, 10, msg, nil)
		st.payer = stvVrc25Contract
		st.initialGas = gasLimit
		st.gas = gasLimit - gasUsed
		st.gasPrice = stvVRC25GasPrice
		st.refundGasVrc25(remaining)
		require.Equal(t, 0, big.NewInt(0).Cmp(statedb.GetBalance(stvVrc25Contract)))
		require.Equal(t, 0, big.NewInt(0).Cmp(statedb.GetBalance(stvSender)))
	})

	t.Run("post_atlas_vrc25_deduct_and_refund", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gasLimit, zeroPrice)
		st, statedb := newVicST(cfg, 150, msg, nil)
		st.payer = stvVrc25Contract
		st.initialGas = gasLimit
		st.gas = gasLimit - gasUsed
		st.gasPrice = stvVRC25GasPrice
		initialCap := big.NewInt(10_000_000_000_000)
		statedb.VicSetZeroGasCapacity(stvVrc25Contract, stvToken, new(big.Int).Set(initialCap))
		st.refundGasVrc25(remaining)
		gasUsedFee := new(big.Int).Mul(new(big.Int).SetUint64(gasUsed), stvVRC25GasPrice)
		expectedBalance := new(big.Int).Sub(initialCap, gasUsedFee)
		require.Equal(t, 0, expectedBalance.Cmp(statedb.VicGetVrc25Balance(stvVrc25Contract, stvToken)))
		require.Equal(t, 0, remaining.Cmp(statedb.GetBalance(stvVrc25Contract)))
	})

	t.Run("post_atlas_normal_refund_to_sender", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gasLimit, zeroPrice)
		st, statedb := newVicST(cfg, 250, msg, nil)
		st.payer = stvSender
		st.initialGas = gasLimit
		st.gas = gasLimit - gasUsed
		st.gasPrice = big.NewInt(1000)
		st.refundGasVrc25(remaining)
		require.Equal(t, 0, remaining.Cmp(statedb.GetBalance(stvSender)))
	})

	t.Run("between_atlas_and_postatlas_normal_burned", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gasLimit, zeroPrice)
		st, statedb := newVicST(cfg, 150, msg, nil)
		st.payer = stvSender
		st.initialGas = gasLimit
		st.gas = gasLimit - gasUsed
		st.gasPrice = big.NewInt(1000)
		st.refundGasVrc25(remaining)
		require.Equal(t, 0, big.NewInt(0).Cmp(statedb.GetBalance(stvSender)))
		require.Equal(t, 0, big.NewInt(0).Cmp(statedb.GetBalance(stvCoinbase)))
	})
}

func TestDistributeFee(t *testing.T) {
	const gas uint64 = 21000
	gasPrice := big.NewInt(1000)
	tokenPtr := &stvToken
	expectedTxFee := new(big.Int).Mul(new(big.Int).SetUint64(gas), gasPrice)
	expectedVrc25TxFee := new(big.Int).Mul(new(big.Int).SetUint64(gas), stvVRC25GasPrice)

	setValidatorOwner := func(statedb *state.StateDB, validator, owner common.Address) {
		slot := state.StorageLocationOfValidatorOwner(validator)
		statedb.SetState(stvValidatorContract, slot, common.BytesToHash(owner.Bytes()))
	}

	t.Run("nil_viction_to_coinbase", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		cfg.Viction = nil
		msg := newVicMsg(stvSender, tokenPtr, gas, gasPrice)
		st, statedb := newVicST(cfg, 10, msg, nil)
		st.initialGas = gas
		st.gas = 0
		st.gasPrice = gasPrice
		st.distributeFee()
		require.Equal(t, 0, expectedTxFee.Cmp(statedb.GetBalance(stvCoinbase)))
	})

	t.Run("nil_tip_gas_price_to_coinbase", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), nil)
		msg := newVicMsg(stvSender, tokenPtr, gas, gasPrice)
		st, statedb := newVicST(cfg, 10, msg, nil)
		st.initialGas = gas
		st.gas = 0
		st.gasPrice = gasPrice
		st.distributeFee()
		require.Equal(t, 0, expectedTxFee.Cmp(statedb.GetBalance(stvCoinbase)))
	})

	t.Run("pre_tip_to_coinbase", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, gasPrice)
		st, statedb := newVicST(cfg, 10, msg, nil)
		st.initialGas = gas
		st.gas = 0
		st.gasPrice = gasPrice
		st.distributeFee()
		require.Equal(t, 0, expectedTxFee.Cmp(statedb.GetBalance(stvCoinbase)))
	})

	t.Run("post_tip_no_owner_burned", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, gasPrice)
		st, statedb := newVicST(cfg, 60, msg, nil)
		st.initialGas = gas
		st.gas = 0
		st.gasPrice = gasPrice
		st.distributeFee()
		require.Equal(t, 0, big.NewInt(0).Cmp(statedb.GetBalance(stvCoinbase)))
		require.Equal(t, 0, big.NewInt(0).Cmp(statedb.GetBalance(stvOwner)))
	})

	t.Run("post_tip_owner_routed", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, gasPrice)
		st, statedb := newVicST(cfg, 60, msg, nil)
		st.initialGas = gas
		st.gas = 0
		st.gasPrice = gasPrice
		setValidatorOwner(statedb, stvCoinbase, stvOwner)
		st.distributeFee()
		require.Equal(t, 0, expectedTxFee.Cmp(statedb.GetBalance(stvOwner)))
		require.Equal(t, 0, big.NewInt(0).Cmp(statedb.GetBalance(stvCoinbase)))
	})

	t.Run("tip_boundary_at_fork_block_to_coinbase", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, gasPrice)
		st, statedb := newVicST(cfg, 50, msg, nil)
		st.initialGas = gas
		st.gas = 0
		st.gasPrice = gasPrice
		setValidatorOwner(statedb, stvCoinbase, stvOwner)
		st.distributeFee()
		require.Equal(t, 0, expectedTxFee.Cmp(statedb.GetBalance(stvCoinbase)))
		require.Equal(t, 0, big.NewInt(0).Cmp(statedb.GetBalance(stvOwner)))
	})

	t.Run("tip_boundary_after_fork_block_to_owner", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, gasPrice)
		st, statedb := newVicST(cfg, 51, msg, nil)
		st.initialGas = gas
		st.gas = 0
		st.gasPrice = gasPrice
		setValidatorOwner(statedb, stvCoinbase, stvOwner)
		st.distributeFee()
		require.Equal(t, 0, expectedTxFee.Cmp(statedb.GetBalance(stvOwner)))
		require.Equal(t, 0, big.NewInt(0).Cmp(statedb.GetBalance(stvCoinbase)))
	})

	t.Run("post_atlas_vrc25_pre_tip_to_coinbase", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(300))
		msg := newVicMsg(stvSender, tokenPtr, gas, big.NewInt(0))
		st, statedb := newVicST(cfg, 150, msg, nil)
		st.payer = stvVrc25Contract
		st.initialGas = gas
		st.gas = 0
		st.gasPrice = big.NewInt(1000)
		st.distributeFee()
		require.Equal(t, 0, expectedVrc25TxFee.Cmp(statedb.GetBalance(stvCoinbase)))
	})

	t.Run("post_atlas_vrc25_post_tip_owner_routed", func(t *testing.T) {
		cfg := vicChainConfig(big.NewInt(100), big.NewInt(200), big.NewInt(50))
		msg := newVicMsg(stvSender, tokenPtr, gas, big.NewInt(0))
		st, statedb := newVicST(cfg, 150, msg, nil)
		st.payer = stvVrc25Contract
		st.initialGas = gas
		st.gas = 0
		st.gasPrice = big.NewInt(1000)
		setValidatorOwner(statedb, stvCoinbase, stvOwner)
		st.distributeFee()
		require.Equal(t, 0, expectedVrc25TxFee.Cmp(statedb.GetBalance(stvOwner)))
		require.Equal(t, 0, big.NewInt(0).Cmp(statedb.GetBalance(stvCoinbase)))
	})
}
