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

package posv

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// finalityThresholdPercent is the share of an electorate whose votes justify a signed block.
	finalityThresholdPercent = 75

	// BlockSign call data is the method selector, then the 32-byte block number, then the 32-byte block hash.
	signDataNumberOffset = 4
	signDataHashOffset   = 36
)

var (
	// errFinalityInactive is returned when the chain config schedules no finality: the Prometheus fork or one of the rule's parameters is unset.
	errFinalityInactive = errors.New("finality is not active")

	// errInvalidFinalityChain is returned when a view is asked for a block that is not the child of the block the view was derived at.
	errInvalidFinalityChain = errors.New("invalid finality chain")

	// errInvalidFinalityRoot is returned when a root view is requested for headers that are not the root and its epoch header.
	errInvalidFinalityRoot = errors.New("invalid finality root")
)

// BlockRef identifies a block by number and hash.
type BlockRef struct {
	Number uint64
	Hash   common.Hash
}

// FinalityTally collects the votes for one target while its vote window is open.
type FinalityTally struct {
	Target BlockRef         // Signed block being voted on
	Source BlockRef         // Highest justified ancestor of the target, as seen at the target
	Epoch  BlockRef         // Epoch header whose validators are the electorate of the target
	Voters []common.Address // Electorate members whose vote was counted, in order of inclusion
}

// FinalityView is the finality state of a chain as seen at one of its blocks.
//
// Validators vote on signed blocks, the blocks whose number is a multiple of the
// sign interval, by including a BlockSign transaction that names the block. The
// rule reads those votes as follows:
//
//   - A vote counts when it names a signed block by number and hash, comes from a
//     member of that block's electorate (the validators in the epoch header on
//     the block's own chain), and is included in the block's vote window, the
//     blocks that follow it. Each member counts once.
//   - A signed block is justified once finalityThresholdPercent of its electorate
//     voted for it.
//   - The source of a signed block is its highest justified ancestor, judged by
//     the votes included up to and including the block itself.
//   - A justified block is finalized when the next signed block is justified and
//     has it as source. Its ancestors are final with it.
//
// Voting starts at the first signed block at or after the Prometheus fork. The
// signed block before that one is the root: it counts as justified and is never
// finalized, so no block becomes final without votes.
//
// A view is not modified once derived: apply returns the view of the next block.
type FinalityView struct {
	Number    uint64           // Block number the view was derived at
	Hash      common.Hash      // Block hash the view was derived at
	Epoch     BlockRef         // Latest epoch header at or below this block
	Justified BlockRef         // Highest justified signed block
	Finalized *BlockRef        // Highest finalized signed block, nil until a block is finalized
	Tallies   []*FinalityTally // Targets inside their vote window and not yet justified, ascending
}

// Return the view at the root of finality: the signed block before the first target.
// The epoch header must be the latest one at or below the root on the root's own chain.
func newRootFinalityView(config *params.ChainConfig, root, epoch *types.Header) (*FinalityView, error) {
	p, err := newFinalityParams(config)
	if err != nil {
		return nil, err
	}
	rootNumber := p.firstTarget - p.interval
	if root.Number.Uint64() != rootNumber {
		return nil, fmt.Errorf("%w: block %d is not the root %d", errInvalidFinalityRoot, root.Number, rootNumber)
	}
	epochNumber := rootNumber - rootNumber%p.epoch
	if epoch.Number.Uint64() != epochNumber {
		return nil, fmt.Errorf("%w: block %d is not the epoch header %d", errInvalidFinalityRoot, epoch.Number, epochNumber)
	}
	if epochNumber == rootNumber && epoch.Hash() != root.Hash() {
		return nil, fmt.Errorf("%w: root %d is an epoch header but another header was given", errInvalidFinalityRoot, rootNumber)
	}
	return &FinalityView{
		Number:    rootNumber,
		Hash:      root.Hash(),
		Epoch:     blockRefOf(epoch),
		Justified: blockRefOf(root),
	}, nil
}

// Derive the view of block, which must be the child of the block the receiver was derived at.
// The receiver is left unchanged.
func (v *FinalityView) apply(chain finalityChain, block *types.Block) (*FinalityView, error) {
	p, err := newFinalityParams(chain.Config())
	if err != nil {
		return nil, err
	}
	number := block.NumberU64()
	if number != v.Number+1 || block.ParentHash() != v.Hash {
		return nil, fmt.Errorf("%w: block %d (%s) does not follow block %d (%s)", errInvalidFinalityChain, number, block.Hash(), v.Number, v.Hash)
	}

	next := v.copy()
	next.Number, next.Hash = number, block.Hash()
	if number%p.epoch == 0 {
		next.Epoch = BlockRef{Number: number, Hash: next.Hash}
	}
	next.dropClosedTallies(number, p.window)
	if err := next.countVotes(chain, p, block); err != nil {
		return nil, err
	}
	// Tally opens after this block's votes are counted, so its source is judged at the target itself.
	if number%p.interval == 0 {
		next.Tallies = append(next.Tallies, &FinalityTally{
			Target: BlockRef{Number: number, Hash: next.Hash},
			Source: next.Justified,
			Epoch:  next.Epoch,
		})
	}
	return next, nil
}

// Return a copy that shares no tally with the receiver.
func (v *FinalityView) copy() *FinalityView {
	cpy := *v
	cpy.Tallies = make([]*FinalityTally, len(v.Tallies))
	for i, tally := range v.Tallies {
		tallyCpy := *tally
		tallyCpy.Voters = append([]common.Address(nil), tally.Voters...)
		cpy.Tallies[i] = &tallyCpy
	}
	return &cpy
}

// Count the votes that block includes for the open tallies, justifying each
// target that reaches the threshold. Called only on a view under construction.
func (v *FinalityView) countVotes(chain finalityChain, p *finalityParams, block *types.Block) error {
	signer := types.MakeSigner(chain.Config(), block.Number())
	electorates := make(map[common.Hash][]common.Address)

	for _, tx := range block.Transactions() {
		if !tx.IsSigningTransaction(p.signContract) {
			continue
		}
		tally := v.tallyOf(tx)
		if tally == nil {
			continue
		}
		voter, err := types.Sender(signer, tx)
		if err != nil {
			// Sender that does not recover is no validator; its transaction casts no vote.
			continue
		}
		electorate, ok := electorates[tally.Epoch.Hash]
		if !ok {
			if electorate, err = electorateOf(chain, tally); err != nil {
				return err
			}
			electorates[tally.Epoch.Hash] = electorate
		}
		if common.IndexOf(electorate, voter) == -1 || common.IndexOf(tally.Voters, voter) != -1 {
			continue
		}
		tally.Voters = append(tally.Voters, voter)
		if isJustifiedBy(len(tally.Voters), len(electorate)) {
			v.justify(tally, p)
		}
	}
	return nil
}

// Drop the tallies whose vote window ended before the block at number.
// Called only on a view under construction.
func (v *FinalityView) dropClosedTallies(number, window uint64) {
	open := v.Tallies[:0]
	for _, tally := range v.Tallies {
		if number <= tally.Target.Number+window {
			open = append(open, tally)
		}
	}
	v.Tallies = open
}

// Close the tally of a target that reached the threshold: the target becomes
// the highest justified block when it is above the current one, and its source
// is finalized when the source is the signed block right below the target.
// Called only on a view under construction.
func (v *FinalityView) justify(tally *FinalityTally, p *finalityParams) {
	open := v.Tallies[:0]
	for _, other := range v.Tallies {
		if other != tally {
			open = append(open, other)
		}
	}
	v.Tallies = open

	if tally.Target.Number > v.Justified.Number {
		v.Justified = tally.Target
	}

	source := tally.Source
	isRoot := source.Number < p.firstTarget
	isAdjacent := source.Number+p.interval == tally.Target.Number
	if isRoot || !isAdjacent {
		return
	}
	if v.Finalized == nil || source.Number > v.Finalized.Number {
		v.Finalized = &source
	}
}

// Return the open tally that a BlockSign transaction votes on, or nil when the
// transaction names no open target. The transaction must pass IsSigningTransaction.
func (v *FinalityView) tallyOf(tx *types.Transaction) *FinalityTally {
	data := tx.Data()
	number := new(big.Int).SetBytes(data[signDataNumberOffset:signDataHashOffset])
	hash := common.BytesToHash(data[signDataHashOffset:])
	if !number.IsUint64() {
		return nil
	}
	for _, tally := range v.Tallies {
		if tally.Target.Number == number.Uint64() && tally.Target.Hash == hash {
			return tally
		}
	}
	return nil
}

// finalityChain is the chain access the finality rule needs.
// It has no lookup by number: every header is resolved through the ancestry of the block under evaluation,
// so a fork is never judged by the canonical chain's headers.
type finalityChain interface {
	Config() *params.ChainConfig
	GetHeader(hash common.Hash, number uint64) *types.Header
}

// finalityParams holds the chain parameters of the finality rule.
type finalityParams struct {
	firstTarget  uint64         // First signed block that can be voted on
	interval     uint64         // Distance between signed blocks
	window       uint64         // Number of blocks after a target in which its votes count
	epoch        uint64         // Distance between epoch headers
	signContract common.Address // Receiver of BlockSign transactions
}

// Read the parameters of the finality rule from the chain config.
// Return errFinalityInactive when the config schedules no finality.
func newFinalityParams(config *params.ChainConfig) (*finalityParams, error) {
	if config == nil || config.Posv == nil || config.Viction == nil {
		return nil, errFinalityInactive
	}
	if config.PrometheusBlock == nil || !config.PrometheusBlock.IsUint64() {
		return nil, errFinalityInactive
	}
	p := &finalityParams{
		interval:     config.Viction.ValidatorSignInterval,
		window:       config.Viction.ConsensusLimitTimeFinality,
		epoch:        config.Posv.Epoch,
		signContract: config.Viction.ValidatorBlockSignContract,
	}
	if p.interval == 0 || p.window == 0 || p.epoch == 0 {
		return nil, errFinalityInactive
	}

	fork := config.PrometheusBlock.Uint64()
	p.firstTarget = fork + (p.interval-fork%p.interval)%p.interval
	if p.firstTarget < fork {
		// Rounding up overflowed: no signed block exists at or after the fork.
		return nil, errFinalityInactive
	}
	if p.firstTarget == 0 {
		// First target needs a signed block below it as root; genesis takes that role.
		p.firstTarget = p.interval
	}
	return p, nil
}

// Return the reference of the block with the given header.
func blockRefOf(header *types.Header) BlockRef {
	return BlockRef{Number: header.Number.Uint64(), Hash: header.Hash()}
}

// Return the validators that may vote on the target of tally: those recorded in the epoch header on the target's own chain.
func electorateOf(chain finalityChain, tally *FinalityTally) ([]common.Address, error) {
	header := chain.GetHeader(tally.Epoch.Hash, tally.Epoch.Number)
	if header == nil {
		return nil, fmt.Errorf("%w: epoch header %d (%s) of signed block %d", errCannotGetCheckpointHeader, tally.Epoch.Number, tally.Epoch.Hash, tally.Target.Number)
	}
	return ExtractValidatorsFromCheckpointHeader(header), nil
}

// Report whether votes out of an electorate of the given size reach the finality threshold. An empty electorate justifies nothing.
func isJustifiedBy(votes, electorateSize int) bool {
	return electorateSize > 0 && votes*100 >= finalityThresholdPercent*electorateSize
}
