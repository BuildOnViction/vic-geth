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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// StateProcessor is a basic Processor, which takes care of transitioning
// state from one point to another.
//
// StateProcessor implements Processor.
type StateProcessor struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for block rewards

	// viction owns all Viction-specific processing hooks (hardfork activation,
	// system transactions, VRC25 fees, native trading/lending replay). See viction.Processor.
	viction *VictionProcessor

	// Deferred trie GC fields for native trading/lending (full-node path).
	// These are managed entirely by blockchain_viction.go / commitVictionState.
	tradingTriegc *prque.Prque // deferred GC queue for native trading trie roots
	lendingTriegc *prque.Prque // deferred GC queue for native lending trie roots
}

// NewStateProcessor initialises a new StateProcessor.
func NewStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *StateProcessor {
	return &StateProcessor{
		config:        config,
		bc:            bc,
		engine:        engine,
		viction:       NewVictionProcessor(config, bc, engine),
		tradingTriegc: prque.New(nil),
		lendingTriegc: prque.New(nil),
	}
}

// Process processes the state changes according to the Ethereum rules by running
// the transaction messages using the statedb and applying any rewards to both
// the processor (coinbase) and any included uncles.
//
// Process returns the receipts and logs accumulated during the process and
// returns the amount of gas that was used in the process. If any of the
// transactions failed to execute due to insufficient gas it will return an error.
func (p *StateProcessor) Process(block *types.Block, statedb *state.StateDB, cfg vm.Config) (types.Receipts, []*types.Log, uint64, error) {
	// Viction hooks
	if err := p.viction.PreBlockProcess(block, statedb); err != nil {
		return nil, nil, 0, err
	}
	var (
		receipts types.Receipts
		usedGas  = new(uint64)
		header   = block.Header()
		allLogs  []*types.Log
		gp       = new(GasPool).AddGas(block.GasLimit())
	)
	// Mutate the block and state according to any hard-fork specs
	if p.config.DAOForkSupport && p.config.DAOForkBlock != nil && p.config.DAOForkBlock.Cmp(block.Number()) == 0 {
		misc.ApplyDAOHardFork(statedb)
	}
	blockContext := NewEVMBlockContext(header, p.bc, nil)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, p.config, cfg)
	// Iterate over and process the individual transactions
	for i, tx := range block.Transactions() {
		msg, err := tx.AsMessage(types.MakeSigner(p.config, header.Number))
		if err != nil {
			return nil, nil, 0, err
		}
		if err := p.viction.PreApplyTransaction(block, tx, msg, statedb); err != nil {
			return nil, nil, 0, err
		}
		statedb.Prepare(tx.Hash(), block.Hash(), i)

		// Apply Viction-specific system transactions (BlockSigner, native trading/lending).
		handled, receipt, _, err, _ := p.viction.ApplyNativeTransaction(tx, header, statedb, usedGas)
		if err != nil {
			return nil, nil, 0, err
		}

		if !handled {
			receipt, err = applyTransaction(msg, p.config, p.bc, nil, gp, statedb, header, tx, usedGas, vmenv, p.viction.ZeroGasPool())
			if err != nil {
				return nil, nil, 0, err
			}
		}

		// Execute Viction-specific post-transaction logic.
		failed := receipt.Status == types.ReceiptStatusFailed
		if err := p.viction.PostApplyTransaction(tx, msg, statedb, receipt.GasUsed, failed); err != nil {
			return nil, nil, 0, err
		}
		receipts = append(receipts, receipt)
		allLogs = append(allLogs, receipt.Logs...)
	}
	// Finalize the block, applying any consensus engine specific extras (e.g. block rewards)
	p.engine.Finalize(p.bc, header, statedb, block.Transactions(), block.Uncles())

	// Execute Viction-specific post-processing logic.
	if err := p.viction.PostBlockProcess(block, statedb); err != nil {
		return nil, nil, 0, err
	}

	return receipts, allLogs, *usedGas, nil
}

func applyTransaction(msg types.Message, config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, evm *vm.EVM, zp types.BalanceMap) (*types.Receipt, error) {
	// Create a new context to be used in the EVM environment
	txContext := NewEVMTxContext(msg)
	// Add addresses to access list if applicable
	if config.IsYoloV2(header.Number) {
		statedb.AddAddressToAccessList(msg.From())
		if dst := msg.To(); dst != nil {
			statedb.AddAddressToAccessList(*dst)
			// If it's a create-tx, the destination will be added inside evm.create
		}
		for _, addr := range evm.ActivePrecompiles() {
			statedb.AddAddressToAccessList(addr)
		}
	}

	// Update the evm with the new transaction context.
	evm.Reset(txContext, statedb)
	// Apply the transaction to the current state (included in the env)
	result, err := ApplyMessage(evm, msg, gp, zp)
	if err != nil {
		return nil, err
	}
	// Update the state with pending changes
	var root []byte
	if config.IsByzantium(header.Number) {
		statedb.Finalise(true)
	} else {
		root = statedb.IntermediateRoot(config.IsEIP158(header.Number)).Bytes()
	}
	*usedGas += result.UsedGas

	// Create a new receipt for the transaction, storing the intermediate root and gas used by the tx
	// based on the eip phase, we're passing whether the root touch-delete accounts.
	receipt := types.NewReceipt(root, result.Failed(), *usedGas)
	receipt.TxHash = tx.Hash()
	receipt.GasUsed = result.UsedGas
	// if the transaction created a contract, store the creation address in the receipt.
	if msg.To() == nil {
		receipt.ContractAddress = crypto.CreateAddress(evm.TxContext.Origin, tx.Nonce())
	}
	// Set the receipt logs and create a bloom for filtering
	receipt.Logs = statedb.GetLogs(tx.Hash())
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	receipt.BlockHash = statedb.BlockHash()
	receipt.BlockNumber = header.Number
	receipt.TransactionIndex = uint(statedb.TxIndex())

	return receipt, err
}

// ApplyTransaction attempts to apply a transaction to the given state database
// and uses the input parameters for its environment. It returns the receipt
// for the transaction, gas used and an error if the transaction failed,
// indicating the block was invalid.
//
// On POSV chains, Viction system transactions (0x89 BlockSigner, 0x91–0x94
// native trading/lending) are handled without the EVM via
// VictionProcessor.ApplyNativeTransaction, the same dispatch used during block
// import. System transactions other than 0x89 do not appear in the tx pool, so
// only the BlockSigner bypass is exercised on this standalone (miner /
// chain_makers) path.
func ApplyTransaction(config *params.ChainConfig, bc ChainContext, author *common.Address, gp *GasPool, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, cfg vm.Config) (*types.Receipt, error) {
	// POSV: Viction system transactions bypass — same logic as block import.
	if handled, receipt, _, err, _ := (&VictionProcessor{config: config}).ApplyNativeTransaction(tx, header, statedb, usedGas); handled {
		return receipt, err
	}

	msg, err := tx.AsMessage(types.MakeSigner(config, header.Number))
	if err != nil {
		return nil, err
	}
	// Create a new context to be used in the EVM environment
	blockContext := NewEVMBlockContext(header, bc, author)
	vmenv := vm.NewEVM(blockContext, vm.TxContext{}, statedb, config, cfg)
	// Standalone path (miner / chain_makers): no per-block fee pool is threaded,
	// so sponsored VRC25 accounting is not applied here. Live blocks are all
	// post-Atlas, where capacity is read directly from state in vrc25BuyGas.
	return applyTransaction(msg, config, bc, author, gp, statedb, header, tx, usedGas, vmenv, nil)
}
