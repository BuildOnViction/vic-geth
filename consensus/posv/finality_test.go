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
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

const (
	finalityTestInterval = 15
	finalityTestWindow   = 30
	finalityTestEpoch    = 60
)

var finalityTestSignContract = common.HexToAddress("0x0000000000000000000000000000000000000089")

func TestFinalityViewFinalizesSourceOfJustifiedChild(t *testing.T) {
	keys := finalityTestKeys(t, 0, 4)
	chain := newFinalityTestChain(0, finalityTestAddresses(keys))
	quorum := keys[:3]

	first := chain.extendTo(t, chain.start(t), 15)
	tip := chain.extend(t, first, chain.votes(t, quorum, first.header)...)
	requireJustified(t, tip.view, first.header)
	requireFinalized(t, tip.view, nil)

	second := chain.extendTo(t, tip, 30)
	tip = chain.extend(t, second, chain.votes(t, quorum, second.header)...)
	requireJustified(t, tip.view, second.header)
	requireFinalized(t, tip.view, first.header)

	third := chain.extendTo(t, tip, 45)
	tip = chain.extend(t, third, chain.votes(t, quorum, third.header)...)
	requireJustified(t, tip.view, third.header)
	requireFinalized(t, tip.view, second.header)
}

func TestFinalityViewNeverFinalizesRoot(t *testing.T) {
	keys := finalityTestKeys(t, 0, 4)
	chain := newFinalityTestChain(20, finalityTestAddresses(keys))
	quorum := keys[:3]

	root := chain.start(t)
	if root.header.Number.Uint64() != 15 {
		t.Fatalf("root number: have %d, want 15", root.header.Number)
	}
	first := chain.extendTo(t, root, 30)
	tip := chain.extend(t, first, chain.votes(t, quorum, first.header)...)
	requireSource(t, first.view, first.header, root.header)
	requireJustified(t, tip.view, first.header)
	requireFinalized(t, tip.view, nil)

	second := chain.extendTo(t, tip, 45)
	tip = chain.extend(t, second, chain.votes(t, quorum, second.header)...)
	requireFinalized(t, tip.view, first.header)
}

func TestFinalityViewCountsOnlyValidVotes(t *testing.T) {
	keys := finalityTestKeys(t, 0, 4)
	outsider := finalityTestKeys(t, 200, 1)[0]

	// Each case alters one field of a vote that would otherwise be the third counted vote for the target.
	tests := []struct {
		name        string
		alter       func(vote *finalityTestVote, sibling *types.Header)
		wantCounted bool
	}{
		{
			name:        "third member is counted",
			alter:       func(vote *finalityTestVote, sibling *types.Header) {},
			wantCounted: true,
		},
		{
			name:  "non-member is ignored",
			alter: func(vote *finalityTestVote, sibling *types.Header) { vote.key = outsider },
		},
		{
			name:  "second vote of one validator is ignored",
			alter: func(vote *finalityTestVote, sibling *types.Header) { vote.key, vote.nonce = keys[0], 1 },
		},
		{
			name:  "vote naming another number is ignored",
			alter: func(vote *finalityTestVote, sibling *types.Header) { vote.number++ },
		},
		{
			name:  "vote naming a block of another fork is ignored",
			alter: func(vote *finalityTestVote, sibling *types.Header) { vote.hash = sibling.Hash() },
		},
		{
			name:  "vote sent to another contract is ignored",
			alter: func(vote *finalityTestVote, sibling *types.Header) { vote.to = common.HexToAddress("0x90") },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := newFinalityTestChain(0, finalityTestAddresses(keys))
			root := chain.start(t)
			parent := chain.extendTo(t, root, 14)
			target := chain.extend(t, parent)
			chain.fork = 1
			sibling := chain.addHeader(parent.header)
			chain.fork = 0
			vote := finalityTestVote{key: keys[2], to: finalityTestSignContract, number: 15, hash: target.header.Hash()}
			tt.alter(&vote, sibling)
			txs := append(chain.votes(t, keys[:2], target.header), chain.voteTx(t, vote))

			tip := chain.extend(t, target, txs...)

			if tt.wantCounted {
				requireJustified(t, tip.view, target.header)
				return
			}
			requireJustified(t, tip.view, root.header)
			requireVoters(t, tip.view, target.header, 2)
		})
	}
}

func TestFinalityViewVoteWindow(t *testing.T) {
	keys := finalityTestKeys(t, 0, 4)

	tests := []struct {
		name          string
		blocksAfter   uint64
		wantJustified bool
	}{
		{name: "vote in the last block of the window is counted", blocksAfter: finalityTestWindow, wantJustified: true},
		{name: "vote in the first block after the window is ignored", blocksAfter: finalityTestWindow + 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := newFinalityTestChain(0, finalityTestAddresses(keys))
			root := chain.start(t)
			target := chain.extendTo(t, root, 15)
			tip := chain.extend(t, target, chain.votes(t, keys[:2], target.header)...)
			tip = chain.extendTo(t, tip, 15+tt.blocksAfter-1)

			tip = chain.extend(t, tip, chain.votes(t, keys[2:3], target.header)...)

			if tt.wantJustified {
				requireJustified(t, tip.view, target.header)
				return
			}
			requireJustified(t, tip.view, root.header)
			if tally := openTally(tip.view, target.header); tally != nil {
				t.Fatalf("tally of block %d is still open after its window", target.header.Number)
			}
		})
	}
}

func TestFinalityViewThreshold(t *testing.T) {
	tests := []struct {
		name          string
		validators    int
		votes         int
		wantJustified bool
	}{
		{name: "112 of 150 votes do not justify", validators: 150, votes: 112},
		{name: "113 of 150 votes justify", validators: 150, votes: 113, wantJustified: true},
		{name: "2 of 4 votes do not justify", validators: 4, votes: 2},
		{name: "3 of 4 votes justify", validators: 4, votes: 3, wantJustified: true},
		{name: "the only validator justifies alone", validators: 1, votes: 1, wantJustified: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := finalityTestKeys(t, 0, tt.validators)
			chain := newFinalityTestChain(0, finalityTestAddresses(keys))
			root := chain.start(t)
			target := chain.extendTo(t, root, 15)

			tip := chain.extend(t, target, chain.votes(t, keys[:tt.votes], target.header)...)

			if tt.wantJustified {
				requireJustified(t, tip.view, target.header)
				return
			}
			requireJustified(t, tip.view, root.header)
			requireVoters(t, tip.view, target.header, tt.votes)
		})
	}
}

func TestIsJustifiedByEmptyElectorate(t *testing.T) {
	if isJustifiedBy(0, 0) {
		t.Fatal("an empty electorate justified a block")
	}
}

func TestFinalityViewSkippedBlockBreaksAdjacency(t *testing.T) {
	keys := finalityTestKeys(t, 0, 4)
	chain := newFinalityTestChain(0, finalityTestAddresses(keys))
	quorum := keys[:3]

	first := chain.extendTo(t, chain.start(t), 15)
	tip := chain.extend(t, first, chain.votes(t, quorum, first.header)...)
	skipped := chain.extendTo(t, tip, 30)
	third := chain.extendTo(t, skipped, 45)
	requireSource(t, third.view, third.header, first.header)

	tip = chain.extend(t, third, chain.votes(t, quorum, third.header)...)
	requireJustified(t, tip.view, third.header)
	requireFinalized(t, tip.view, nil)

	fourth := chain.extendTo(t, tip, 60)
	tip = chain.extend(t, fourth, chain.votes(t, quorum, fourth.header)...)
	requireFinalized(t, tip.view, third.header)
}

func TestFinalityViewJudgesSourceAtTarget(t *testing.T) {
	keys := finalityTestKeys(t, 0, 4)
	chain := newFinalityTestChain(0, finalityTestAddresses(keys))
	quorum := keys[:3]

	root := chain.start(t)
	first := chain.extendTo(t, root, 15)
	second := chain.extendTo(t, first, 30)
	requireSource(t, second.view, second.header, root.header)

	tip := chain.extend(t, second, chain.votes(t, quorum, first.header)...)
	requireJustified(t, tip.view, first.header)
	tip = chain.extend(t, tip, chain.votes(t, quorum, second.header)...)
	requireJustified(t, tip.view, second.header)
	requireFinalized(t, tip.view, nil)

	third := chain.extendTo(t, tip, 45)
	tip = chain.extend(t, third, chain.votes(t, quorum, third.header)...)
	requireFinalized(t, tip.view, second.header)
}

func TestFinalityViewCountsVotesInTargetBlockForItsSource(t *testing.T) {
	keys := finalityTestKeys(t, 0, 4)
	chain := newFinalityTestChain(0, finalityTestAddresses(keys))
	quorum := keys[:3]
	first := chain.extendTo(t, chain.start(t), 15)
	tip := chain.extendTo(t, first, 29)

	second := chain.extend(t, tip, chain.votes(t, quorum, first.header)...)

	requireJustified(t, second.view, first.header)
	requireSource(t, second.view, second.header, first.header)
	tip = chain.extend(t, second, chain.votes(t, quorum, second.header)...)
	requireFinalized(t, tip.view, first.header)
}

func TestFinalityViewLateJustificationNeverLowersView(t *testing.T) {
	keys := finalityTestKeys(t, 0, 4)
	chain := newFinalityTestChain(0, finalityTestAddresses(keys))
	chain.config.Viction.ConsensusLimitTimeFinality = 3 * finalityTestInterval
	quorum := keys[:3]
	first := chain.extendTo(t, chain.start(t), 15)
	tip := chain.extend(t, first, chain.votes(t, quorum, first.header)...)
	late := chain.extendTo(t, tip, 30)
	third := chain.extendTo(t, late, 45)
	tip = chain.extend(t, third, chain.votes(t, quorum, third.header)...)
	fourth := chain.extendTo(t, tip, 60)
	tip = chain.extend(t, fourth, chain.votes(t, quorum, fourth.header)...)
	requireJustified(t, tip.view, fourth.header)
	requireFinalized(t, tip.view, third.header)
	requireSource(t, tip.view, late.header, first.header)

	tip = chain.extend(t, tip, chain.votes(t, quorum, late.header)...)

	requireJustified(t, tip.view, fourth.header)
	requireFinalized(t, tip.view, third.header)
	if tally := openTally(tip.view, late.header); tally != nil {
		t.Fatalf("tally of block %d is still open after it was justified", late.header.Number)
	}
}

func TestFinalityViewForksAreIndependent(t *testing.T) {
	keys := finalityTestKeys(t, 0, 4)
	chain := newFinalityTestChain(0, finalityTestAddresses(keys))
	quorum := keys[:3]
	first := chain.extendTo(t, chain.start(t), 15)
	shared := chain.extend(t, first, chain.votes(t, quorum, first.header)...)

	secondA := chain.extendTo(t, shared, 30)
	tipA := chain.extend(t, secondA, chain.votes(t, quorum, secondA.header)...)
	chain.fork = 1
	secondB := chain.extendTo(t, shared, 30)
	tipB := chain.extend(t, secondB, chain.votes(t, quorum, secondA.header)...)

	requireJustified(t, tipA.view, secondA.header)
	requireFinalized(t, tipA.view, first.header)
	requireJustified(t, tipB.view, first.header)
	requireFinalized(t, tipB.view, nil)
	requireVoters(t, tipB.view, secondB.header, 0)
	requireJustified(t, shared.view, first.header)
	requireFinalized(t, shared.view, nil)
}

func TestFinalityViewApplyLeavesOtherViewsUnchanged(t *testing.T) {
	keys := finalityTestKeys(t, 0, 8)
	addresses := finalityTestAddresses(keys)
	chain := newFinalityTestChain(0, addresses)
	target := chain.extendTo(t, chain.start(t), 15)
	parent := target
	for _, key := range keys[:3] {
		parent = chain.extend(t, parent, chain.votes(t, []*ecdsa.PrivateKey{key}, target.header)...)
	}

	childA := chain.extend(t, parent, chain.votes(t, keys[3:4], target.header)...)
	chain.fork = 1
	childB := chain.extend(t, parent, chain.votes(t, keys[4:5], target.header)...)

	requireVoterList(t, parent.view, target.header, addresses[:3])
	requireVoterList(t, childA.view, target.header, addresses[:4])
	requireVoterList(t, childB.view, target.header, append(append([]common.Address(nil), addresses[:3]...), addresses[4]))
	if parent.view.Number != 18 || parent.view.Hash != parent.header.Hash() {
		t.Fatalf("parent view moved to block %d (%s)", parent.view.Number, parent.view.Hash)
	}
}

func TestFinalityViewOpensTalliesOnlyAtSignedBlocks(t *testing.T) {
	chain := newFinalityTestChain(0, nil)
	root := chain.start(t)

	beforeFirst := chain.extendTo(t, root, 14)
	afterSecond := chain.extendTo(t, beforeFirst, 31)

	if len(root.view.Tallies) != 0 || len(beforeFirst.view.Tallies) != 0 {
		t.Fatalf("open tallies before the first signed block: have %d at the root and %d at block 14, want 0", len(root.view.Tallies), len(beforeFirst.view.Tallies))
	}
	if len(afterSecond.view.Tallies) != 2 {
		t.Fatalf("open tallies at block 31: have %d, want 2", len(afterSecond.view.Tallies))
	}
	for i, want := range []uint64{15, 30} {
		if have := afterSecond.view.Tallies[i].Target.Number; have != want {
			t.Fatalf("target of tally %d: have %d, want %d", i, have, want)
		}
	}
}

func TestFinalityViewElectorateFollowsEpochHeader(t *testing.T) {
	outgoing := finalityTestKeys(t, 0, 4)
	incoming := finalityTestKeys(t, 4, 4)
	chain := newFinalityTestChain(0, finalityTestAddresses(outgoing))
	root := chain.start(t)
	before := chain.extendTo(t, root, 45)
	chain.validators = finalityTestAddresses(incoming)
	boundary := chain.extendTo(t, before, finalityTestEpoch)

	wrongSets := append(chain.votes(t, incoming, before.header), chain.votes(t, outgoing, boundary.header)...)
	tip := chain.extend(t, boundary, wrongSets...)
	requireJustified(t, tip.view, root.header)
	requireVoters(t, tip.view, before.header, 0)
	requireVoters(t, tip.view, boundary.header, 0)

	tip = chain.extend(t, tip, chain.votes(t, outgoing[:3], before.header)...)
	requireJustified(t, tip.view, before.header)

	tip = chain.extend(t, tip, chain.votes(t, incoming[:3], boundary.header)...)
	requireJustified(t, tip.view, boundary.header)
}

func TestFinalityViewApplyErrors(t *testing.T) {
	keys := finalityTestKeys(t, 0, 4)

	tests := []struct {
		name    string
		apply   func(chain *finalityTestChain, target finalityTestTip) error
		wantErr error
	}{
		{
			name: "block that does not follow the view",
			apply: func(chain *finalityTestChain, target finalityTestTip) error {
				grandchild := chain.addHeader(chain.addHeader(target.header))
				_, err := target.view.apply(chain, types.NewBlockWithHeader(grandchild))
				return err
			},
			wantErr: errInvalidFinalityChain,
		},
		{
			name: "child whose number skips one",
			apply: func(chain *finalityTestChain, target finalityTestTip) error {
				child := types.CopyHeader(chain.addHeader(target.header))
				child.Number.Add(child.Number, big.NewInt(1))
				_, err := target.view.apply(chain, types.NewBlockWithHeader(child))
				return err
			},
			wantErr: errInvalidFinalityChain,
		},
		{
			name: "sibling of the block the view was derived at",
			apply: func(chain *finalityTestChain, target finalityTestTip) error {
				chain.fork = 1
				sibling := chain.addHeader(chain.headers[target.header.ParentHash])
				_, err := target.view.apply(chain, types.NewBlockWithHeader(chain.addHeader(sibling)))
				return err
			},
			wantErr: errInvalidFinalityChain,
		},
		{
			name: "epoch header missing from the chain",
			apply: func(chain *finalityTestChain, target finalityTestTip) error {
				delete(chain.headers, target.view.Epoch.Hash)
				block := types.NewBlockWithHeader(chain.addHeader(target.header)).WithBody(chain.votes(t, keys[:1], target.header), nil)
				_, err := target.view.apply(chain, block)
				return err
			},
			wantErr: errCannotGetCheckpointHeader,
		},
		{
			name: "config without the Prometheus fork",
			apply: func(chain *finalityTestChain, target finalityTestTip) error {
				chain.config.PrometheusBlock = nil
				_, err := target.view.apply(chain, types.NewBlockWithHeader(chain.addHeader(target.header)))
				return err
			},
			wantErr: errFinalityInactive,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := newFinalityTestChain(0, finalityTestAddresses(keys))
			target := chain.extendTo(t, chain.start(t), 15)

			err := tt.apply(chain, target)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error: have %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewFinalityParamsFirstTarget(t *testing.T) {
	tests := []struct {
		name       string
		prometheus uint64
		want       uint64
	}{
		{name: "fork at genesis starts at the first signed block above it", prometheus: 0, want: 15},
		{name: "fork below a signed block rounds up", prometheus: 1, want: 15},
		{name: "fork on a signed block starts there", prometheus: 15, want: 15},
		{name: "fork right above a signed block rounds up to the next", prometheus: 16, want: 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := newFinalityTestChain(tt.prometheus, nil)

			p, err := newFinalityParams(chain.config)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.firstTarget != tt.want {
				t.Fatalf("first target: have %d, want %d", p.firstTarget, tt.want)
			}
		})
	}
}

func TestNewFinalityParamsInactive(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(config *params.ChainConfig)
	}{
		{name: "no Prometheus fork", mutate: func(config *params.ChainConfig) { config.PrometheusBlock = nil }},
		{name: "negative Prometheus fork", mutate: func(config *params.ChainConfig) { config.PrometheusBlock = big.NewInt(-1) }},
		{name: "no PoSV config", mutate: func(config *params.ChainConfig) { config.Posv = nil }},
		{name: "no Viction config", mutate: func(config *params.ChainConfig) { config.Viction = nil }},
		{name: "zero sign interval", mutate: func(config *params.ChainConfig) { config.Viction.ValidatorSignInterval = 0 }},
		{name: "zero vote window", mutate: func(config *params.ChainConfig) { config.Viction.ConsensusLimitTimeFinality = 0 }},
		{name: "zero epoch", mutate: func(config *params.ChainConfig) { config.Posv.Epoch = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := newFinalityTestChain(0, nil).config
			tt.mutate(config)

			_, err := newFinalityParams(config)

			if !errors.Is(err, errFinalityInactive) {
				t.Fatalf("error: have %v, want %v", err, errFinalityInactive)
			}
		})
	}
	t.Run("nil config", func(t *testing.T) {
		if _, err := newFinalityParams(nil); !errors.Is(err, errFinalityInactive) {
			t.Fatalf("error: have %v, want %v", err, errFinalityInactive)
		}
	})
}

func TestNewRootFinalityView(t *testing.T) {
	chain := newFinalityTestChain(20, nil)
	genesis := chain.addHeader(nil)
	root := genesis
	for root.Number.Uint64() < 15 {
		root = chain.addHeader(root)
	}

	view, err := newRootFinalityView(chain.config, root, genesis)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if view.Number != 15 || view.Hash != root.Hash() {
		t.Fatalf("view block: have %d (%s), want 15 (%s)", view.Number, view.Hash, root.Hash())
	}
	if view.Epoch != blockRefOf(genesis) {
		t.Fatalf("epoch header: have %d, want 0", view.Epoch.Number)
	}
	requireJustified(t, view, root)
	requireFinalized(t, view, nil)
	if len(view.Tallies) != 0 {
		t.Fatalf("open tallies: have %d, want 0", len(view.Tallies))
	}
}

func TestNewRootFinalityViewRejectsWrongHeaders(t *testing.T) {
	tests := []struct {
		name       string
		prometheus uint64
		headers    func(chain *finalityTestChain, byNumber map[uint64]*types.Header) (root, epoch *types.Header)
	}{
		{
			name:       "header below the root",
			prometheus: 20,
			headers: func(chain *finalityTestChain, byNumber map[uint64]*types.Header) (root, epoch *types.Header) {
				return byNumber[14], byNumber[0]
			},
		},
		{
			name:       "header that is not the epoch header",
			prometheus: 20,
			headers: func(chain *finalityTestChain, byNumber map[uint64]*types.Header) (root, epoch *types.Header) {
				return byNumber[15], byNumber[15]
			},
		},
		{
			name:       "epoch header of another fork at the root's number",
			prometheus: 75,
			headers: func(chain *finalityTestChain, byNumber map[uint64]*types.Header) (root, epoch *types.Header) {
				chain.fork = 1
				return byNumber[60], chain.addHeader(byNumber[59])
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := newFinalityTestChain(tt.prometheus, nil)
			byNumber := map[uint64]*types.Header{0: chain.addHeader(nil)}
			for number := uint64(1); number <= finalityTestEpoch; number++ {
				byNumber[number] = chain.addHeader(byNumber[number-1])
			}
			root, epoch := tt.headers(chain, byNumber)

			_, err := newRootFinalityView(chain.config, root, epoch)

			if !errors.Is(err, errInvalidFinalityRoot) {
				t.Fatalf("error: have %v, want %v", err, errInvalidFinalityRoot)
			}
		})
	}
}

// finalityTestChain is an in-memory set of headers addressed by hash, so forks can coexist.
type finalityTestChain struct {
	config     *params.ChainConfig
	blocks     map[common.Hash]*types.Block // Blocks added by extend, read back when a view is derived by replay
	headers    map[common.Hash]*types.Header
	head       *types.Header    // Head of the canonical chain, which lookups by number follow
	validators []common.Address // Written into every epoch header added from now on
	fork       byte             // Tells sibling headers apart
}

// finalityTestTip is a chain head under test: a header and the view derived at it.
type finalityTestTip struct {
	header *types.Header
	view   *FinalityView
}

// finalityTestVote describes a BlockSign transaction to build.
type finalityTestVote struct {
	key    *ecdsa.PrivateKey
	nonce  uint64
	to     common.Address
	number uint64
	hash   common.Hash
}

func newFinalityTestChain(prometheus uint64, validators []common.Address) *finalityTestChain {
	return &finalityTestChain{
		config: &params.ChainConfig{
			PrometheusBlock: new(big.Int).SetUint64(prometheus),
			Posv:            &params.PosvConfig{Period: 2, Epoch: finalityTestEpoch, Gap: 5},
			Viction: &params.VictionConfig{
				ConsensusLimitTimeFinality: finalityTestWindow,
				ValidatorBlockSignContract: finalityTestSignContract,
				ValidatorSignInterval:      finalityTestInterval,
			},
		},
		blocks:     make(map[common.Hash]*types.Block),
		headers:    make(map[common.Hash]*types.Header),
		validators: validators,
	}
}

func (c *finalityTestChain) Config() *params.ChainConfig {
	return c.config
}

func (c *finalityTestChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	header := c.headers[hash]
	if header == nil || header.Number.Uint64() != number {
		return nil
	}
	return header
}

// addHeader stores and returns a child of parent, or a genesis header when parent is nil.
func (c *finalityTestChain) addHeader(parent *types.Header) *types.Header {
	header := &types.Header{Number: new(big.Int), Difficulty: big.NewInt(1)}
	if parent != nil {
		header.Number.Add(parent.Number, big.NewInt(1))
		header.ParentHash = parent.Hash()
	}
	extra := make([]byte, ExtraVanity)
	extra[0] = c.fork
	if header.Number.Uint64()%finalityTestEpoch == 0 {
		for _, validator := range c.validators {
			extra = append(extra, validator.Bytes()...)
		}
	}
	header.Extra = append(extra, make([]byte, ExtraSeal)...)
	c.headers[header.Hash()] = header
	return header
}

// start builds the chain from genesis to the root and returns the tip at the root.
func (c *finalityTestChain) start(t *testing.T) finalityTestTip {
	t.Helper()
	p, err := newFinalityParams(c.config)
	if err != nil {
		t.Fatalf("finality params: %v", err)
	}
	if p.firstTarget < p.interval {
		t.Fatalf("first target %d leaves no signed block below it as root", p.firstTarget)
	}
	header := c.addHeader(nil)
	epoch := header
	for header.Number.Uint64() < p.firstTarget-p.interval {
		header = c.addHeader(header)
		if header.Number.Uint64()%p.epoch == 0 {
			epoch = header
		}
	}
	view, err := newRootFinalityView(c.config, header, epoch)
	if err != nil {
		t.Fatalf("root view: %v", err)
	}
	return finalityTestTip{header: header, view: view}
}

// extend adds one block that includes txs on top of tip.
func (c *finalityTestChain) extend(t *testing.T, tip finalityTestTip, txs ...*types.Transaction) finalityTestTip {
	t.Helper()
	header := c.addHeader(tip.header)
	block := types.NewBlockWithHeader(header).WithBody(txs, nil)
	c.blocks[block.Hash()] = block
	view, err := tip.view.apply(c, block)
	if err != nil {
		t.Fatalf("apply block %d: %v", header.Number, err)
	}
	return finalityTestTip{header: header, view: view}
}

// extendTo adds empty blocks on top of tip until the block at number.
func (c *finalityTestChain) extendTo(t *testing.T, tip finalityTestTip, number uint64) finalityTestTip {
	t.Helper()
	for tip.header.Number.Uint64() < number {
		tip = c.extend(t, tip)
	}
	return tip
}

// voteTx builds and signs the BlockSign transaction that vote describes.
func (c *finalityTestChain) voteTx(t *testing.T, vote finalityTestVote) *types.Transaction {
	t.Helper()
	data := common.Hex2Bytes("e341eaa4") // sign(uint256,bytes32)
	data = append(data, common.LeftPadBytes(new(big.Int).SetUint64(vote.number).Bytes(), 32)...)
	data = append(data, vote.hash.Bytes()...)
	tx := types.NewTransaction(vote.nonce, vote.to, new(big.Int), 200000, new(big.Int), data)
	signed, err := types.SignTx(tx, types.MakeSigner(c.config, new(big.Int)), vote.key)
	if err != nil {
		t.Fatalf("sign vote: %v", err)
	}
	return signed
}

// votes builds one vote for target from each key.
func (c *finalityTestChain) votes(t *testing.T, keys []*ecdsa.PrivateKey, target *types.Header) []*types.Transaction {
	t.Helper()
	txs := make([]*types.Transaction, len(keys))
	for i, key := range keys {
		txs[i] = c.voteTx(t, finalityTestVote{key: key, to: finalityTestSignContract, number: target.Number.Uint64(), hash: target.Hash()})
	}
	return txs
}

// finalityTestKeys returns count deterministic keys, numbered from first.
func finalityTestKeys(t *testing.T, first, count int) []*ecdsa.PrivateKey {
	t.Helper()
	keys := make([]*ecdsa.PrivateKey, count)
	for i := range keys {
		key, err := crypto.HexToECDSA(fmt.Sprintf("%064x", first+i+1))
		if err != nil {
			t.Fatalf("key %d: %v", first+i, err)
		}
		keys[i] = key
	}
	return keys
}

func finalityTestAddresses(keys []*ecdsa.PrivateKey) []common.Address {
	addresses := make([]common.Address, len(keys))
	for i, key := range keys {
		addresses[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	return addresses
}

// openTally returns the open tally of target in view, or nil when there is none.
func openTally(view *FinalityView, target *types.Header) *FinalityTally {
	for _, tally := range view.Tallies {
		if tally.Target == blockRefOf(target) {
			return tally
		}
	}
	return nil
}

func requireJustified(t *testing.T, view *FinalityView, want *types.Header) {
	t.Helper()
	if view.Justified != blockRefOf(want) {
		t.Fatalf("justified block at %d: have %d (%s), want %d (%s)", view.Number, view.Justified.Number, view.Justified.Hash, want.Number, want.Hash())
	}
}

// requireFinalized fails unless view finalized want; a nil want expects no finalized block.
func requireFinalized(t *testing.T, view *FinalityView, want *types.Header) {
	t.Helper()
	switch {
	case want == nil && view.Finalized != nil:
		t.Fatalf("finalized block at %d: have %d, want none", view.Number, view.Finalized.Number)
	case want != nil && view.Finalized == nil:
		t.Fatalf("finalized block at %d: have none, want %d", view.Number, want.Number)
	case want != nil && *view.Finalized != blockRefOf(want):
		t.Fatalf("finalized block at %d: have %d (%s), want %d (%s)", view.Number, view.Finalized.Number, view.Finalized.Hash, want.Number, want.Hash())
	}
}

func requireSource(t *testing.T, view *FinalityView, target, want *types.Header) {
	t.Helper()
	tally := openTally(view, target)
	if tally == nil {
		t.Fatalf("no open tally for block %d at %d", target.Number, view.Number)
	}
	if tally.Source != blockRefOf(want) {
		t.Fatalf("source of block %d: have %d, want %d", target.Number, tally.Source.Number, want.Number)
	}
}

func requireVoters(t *testing.T, view *FinalityView, target *types.Header, want int) {
	t.Helper()
	tally := openTally(view, target)
	if tally == nil {
		t.Fatalf("no open tally for block %d at %d", target.Number, view.Number)
	}
	if len(tally.Voters) != want {
		t.Fatalf("counted voters of block %d: have %d, want %d", target.Number, len(tally.Voters), want)
	}
}

func requireVoterList(t *testing.T, view *FinalityView, target *types.Header, want []common.Address) {
	t.Helper()
	requireVoters(t, view, target, len(want))
	for i, voter := range openTally(view, target).Voters {
		if voter != want[i] {
			t.Fatalf("voter %d of block %d at %d: have %s, want %s", i, target.Number, view.Number, voter, want[i])
		}
	}
}
