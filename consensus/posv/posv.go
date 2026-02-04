// Copyright 2016 The go-ethereum Authors
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

// Package posv implements the proof-of-stake-voting consensus engine.
package posv

import (
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posv/rewards"
	"github.com/ethereum/go-ethereum/consensus/posv/transactions"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	lru "github.com/hashicorp/golang-lru"
)

const (
	inmemorySnapshots      = 128 // Number of recent vote snapshots to keep in memory
	blockSignersCacheLimit = 9000
)

// Posv proof-of-stake-voting protocol constants.
var (
	epochLength = uint64(900) // Default number of blocks after which to checkpoint and reset the pending votes
)

// SignerFn is a signer callback function to request a hash to be signed by a
// backing account.
type SignerFn func(accounts.Account, []byte) ([]byte, error)

// Posv is the proof-of-stake-voting consensus engine.
type Posv struct {
	config *params.PosvConfig // Consensus engine configuration parameters
	db     ethdb.Database     // Database to store and retrieve snapshot checkpoints

	recents    *lru.ARCCache // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache // Signatures of recent blocks to speed up mining

	proposals map[common.Address]bool // Current list of proposals we are pushing

	signer common.Address // Ethereum address of the signing key
	signFn SignerFn       // Signer function to authorize hashes with
	lock   sync.RWMutex   // Protects the signer fields

	rewardCalculator *rewards.RewardCalculator // Modular reward logic

	BlockSigners *lru.Cache
}

// New creates a PoSV proof-of-stake-voting consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.PosvConfig, db ethdb.Database) *Posv {
	// Set any missing consensus parameters to their defaults
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = epochLength
	}
	// Allocate the snapshot caches and create the engine
	BlockSigners, _ := lru.New(blockSignersCacheLimit)
	recents, _ := lru.NewARC(inmemorySnapshots)
	signatures, _ := lru.NewARC(inmemorySnapshots)
	return &Posv{
		config:           &conf,
		db:               db,
		BlockSigners:     BlockSigners,
		recents:          recents,
		signatures:       signatures,
		proposals:        make(map[common.Address]bool),
		rewardCalculator: rewards.NewRewardCalculator(conf.Epoch),
	}
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (c *Posv) Author(header *types.Header) (common.Address, error) {
	// TODO: Implement ecrecover logic
	return common.Address{}, errors.New("not implemented")
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *Posv) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	// TODO: Implement header verification
	return nil
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (c *Posv) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for _, header := range headers {
			err := c.VerifyHeader(chain, header, false)
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (c *Posv) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (c *Posv) VerifySeal(chain consensus.ChainHeaderReader, header *types.Header) error {
	// TODO: Implement seal verification
	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *Posv) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	// TODO: Implement header preparation
	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given.
func (c *Posv) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	// Apply Epoch Rewards (System Logic)
	if err := c.rewardCalculator.ApplyEpochRewards(header, state); err != nil {
		// log.Error("Failed to apply epoch rewards", "err", err)
	}

	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)
}

// FinalizeAndAssemble implements consensus.Engine, ensuring no uncles are set,
// nor block rewards given, and returns the final block.
func (c *Posv) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// TODO: Implement finalization and assembly
	c.Finalize(chain, header, state, txs, uncles)
	return types.NewBlock(header, txs, nil, receipts, nil), nil
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (c *Posv) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	// TODO: Implement sealing logic
	return nil
}

// SealHash returns the hash of a block prior to it being sealed.
func (c *Posv) SealHash(header *types.Header) common.Hash {
	// TODO: Implement seal hash calculation
	return header.Hash()
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (c *Posv) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	// TODO: Implement difficulty calculation
	return big.NewInt(1)
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *Posv) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	// TODO: Implement RPC APIs
	return []rpc.API{}
}

// Close implements consensus.Engine. It's a noop for posv as there are no background threads.
func (c *Posv) Close() error {
	return nil
}

// ProcessSpecificTransaction implements consensus.SpecificTransactionEngine.
func (c *Posv) ProcessSpecificTransaction(state *state.StateDB, tx *types.Transaction, header *types.Header) (bool, *types.Receipt, error) {
	return transactions.ProcessSignTransaction(state, tx, header)
}
