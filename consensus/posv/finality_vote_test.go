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
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

var (
	finalityTestVoter      = common.HexToAddress("0x00000000000000000000000000000000000000a1")
	finalityTestOtherVoter = common.HexToAddress("0x00000000000000000000000000000000000000a2")
)

func TestAuthorizeVoteAppliesSigningRule(t *testing.T) {
	type vote struct {
		fork   string // Fork that holds the target
		number uint64
		head   string // Fork that is canonical when the vote is asked for
	}
	tests := []struct {
		name  string
		first *vote // Vote cast before the one under test, nil for none
		vote  vote
		want  bool
	}{
		{
			name: "first vote is allowed",
			vote: vote{fork: "main", number: 30, head: "main"},
			want: true,
		},
		{
			name:  "next target with a higher source is allowed",
			first: &vote{fork: "main", number: 30, head: "main"},
			vote:  vote{fork: "main", number: 45, head: "main"},
			want:  true,
		},
		{
			name:  "higher target with the same source is allowed",
			first: &vote{fork: "quiet", number: 30, head: "quiet"},
			vote:  vote{fork: "quiet", number: 45, head: "quiet"},
			want:  true,
		},
		{
			name:  "target that skips a height is allowed",
			first: &vote{fork: "main", number: 30, head: "main"},
			vote:  vote{fork: "main", number: 60, head: "main"},
			want:  true,
		},
		{
			name:  "second target at the same height is refused",
			first: &vote{fork: "main", number: 30, head: "main"},
			vote:  vote{fork: "quiet", number: 30, head: "quiet"},
		},
		{
			name:  "same target again is refused",
			first: &vote{fork: "main", number: 30, head: "main"},
			vote:  vote{fork: "main", number: 30, head: "main"},
		},
		{
			name:  "target below the last one is refused",
			first: &vote{fork: "quiet", number: 45, head: "quiet"},
			vote:  vote{fork: "quiet", number: 30, head: "quiet"},
		},
		{
			name:  "higher target with a lower source is refused",
			first: &vote{fork: "main", number: 30, head: "main"},
			vote:  vote{fork: "bare", number: 45, head: "bare"},
		},
		{
			name: "target on a side chain is refused",
			vote: vote{fork: "bare", number: 30, head: "main"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, forks := newFinalityVoteTestForks(t)
			db := rawdb.NewMemoryDatabase()
			engine := New(chain.config.Posv, db)
			var lastTarget *types.Header
			if tt.first != nil {
				lastTarget = forks[tt.first.fork][tt.first.number].header
				chain.head = forks[tt.first.head][62].header
				requireVoteDecision(t, engine, chain, lastTarget, finalityTestVoter, true)
			}
			target := forks[tt.vote.fork][tt.vote.number].header
			chain.head = forks[tt.vote.head][62].header

			isAuthorized, err := engine.AuthorizeVote(chain, target, finalityTestVoter)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if isAuthorized != tt.want {
				t.Fatalf("decision on block %d: have %v, want %v", target.Number, isAuthorized, tt.want)
			}
			if tt.want {
				lastTarget = target
			}
			requireLastVoteTarget(t, db, finalityTestVoter, lastTarget)
		})
	}
}

func TestAuthorizeVoteRecordsSourceOfTarget(t *testing.T) {
	chain, forks := newFinalityVoteTestForks(t)
	db := rawdb.NewMemoryDatabase()
	engine := New(chain.config.Posv, db)
	chain.head = forks["main"][62].header

	requireVoteDecision(t, engine, chain, forks["main"][45].header, finalityTestVoter, true)

	vote, err := loadFinalityVote(db, finalityTestVoter)
	if err != nil || vote == nil {
		t.Fatalf("last vote: have %v (error %v), want a vote", vote, err)
	}
	if vote.Target != blockRefOf(forks["main"][45].header) {
		t.Fatalf("target of last vote: have %d (%s), want block 45 of the main fork", vote.Target.Number, vote.Target.Hash)
	}
	if vote.Source != blockRefOf(forks["main"][30].header) {
		t.Fatalf("source of last vote: have %d (%s), want block 30 of the main fork", vote.Source.Number, vote.Source.Hash)
	}
}

func TestAuthorizeVoteNeedsNoPermissionOutsideTargets(t *testing.T) {
	prometheus := func(number uint64) *uint64 { return &number }
	tests := []struct {
		name       string
		prometheus *uint64 // nil for a chain that schedules no Prometheus fork
		number     uint64
	}{
		{name: "signed block before the first target", prometheus: prometheus(100), number: 90},
		{name: "block that is not a signed block", prometheus: prometheus(0), number: 31},
		{name: "signed block of a chain without the fork", number: 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := newFinalityTestChain(0, nil)
			chain.config.PrometheusBlock = nil
			if tt.prometheus != nil {
				chain.config.PrometheusBlock = new(big.Int).SetUint64(*tt.prometheus)
			}
			db := rawdb.NewMemoryDatabase()
			engine := New(chain.config.Posv, db)
			header := chain.addHeader(nil)
			for header.Number.Uint64() < tt.number {
				header = chain.addHeader(header)
			}

			// No head is set, so the block is on no canonical chain, and it is asked for twice: a target would be refused both times.
			requireVoteDecision(t, engine, chain, header, finalityTestVoter, true)
			requireVoteDecision(t, engine, chain, header, finalityTestVoter, true)

			requireLastVoteTarget(t, db, finalityTestVoter, nil)
		})
	}
}

func TestAuthorizeVoteRecordSurvivesRestart(t *testing.T) {
	chain, forks := newFinalityVoteTestForks(t)
	db := rawdb.NewMemoryDatabase()
	chain.head = forks["main"][62].header
	requireVoteDecision(t, New(chain.config.Posv, db), chain, forks["main"][30].header, finalityTestVoter, true)

	restarted := New(chain.config.Posv, db)

	chain.head = forks["quiet"][62].header
	requireVoteDecision(t, restarted, chain, forks["quiet"][30].header, finalityTestVoter, false)
	chain.head = forks["main"][62].header
	requireVoteDecision(t, restarted, chain, forks["main"][45].header, finalityTestVoter, true)
}

func TestAuthorizeVoteKeepsVotersApart(t *testing.T) {
	chain, forks := newFinalityVoteTestForks(t)
	db := rawdb.NewMemoryDatabase()
	engine := New(chain.config.Posv, db)
	chain.head = forks["main"][62].header
	requireVoteDecision(t, engine, chain, forks["main"][45].header, finalityTestVoter, true)

	requireVoteDecision(t, engine, chain, forks["main"][30].header, finalityTestOtherVoter, true)

	requireLastVoteTarget(t, db, finalityTestVoter, forks["main"][45].header)
	requireLastVoteTarget(t, db, finalityTestOtherVoter, forks["main"][30].header)
}

func TestAuthorizeVoteRefusesUnknownSourceAfterKnownOne(t *testing.T) {
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	db := rawdb.NewMemoryDatabase()
	engine := New(chain.config.Posv, db)
	// Only block 15 is ever justified, so block 30 has it as source, and so has block 195 for a node that knows the full history.
	tips := chain.tipsTo(t, chain.start(t), 200, quorum, func(target uint64) bool { return target == 15 })
	chain.head = tips[200].header
	requireVoteDecision(t, engine, chain, tips[30].header, finalityTestVoter, true)
	requireJustified(t, tips[195].view, tips[15].header)

	// View of block 195 starts unanchored one epoch below it and sees no vote, so its source is unknown.
	requireVoteDecision(t, engine, chain, tips[195].header, finalityTestVoter, false)

	requireLastVoteTarget(t, db, finalityTestVoter, tips[30].header)
}

func TestAuthorizeVoteFailsWithoutRecording(t *testing.T) {
	tests := []struct {
		name    string
		arrange func(t *testing.T, chain *finalityTestChain, db ethdb.Database, head *types.Header)
		wantErr error // nil for any error
	}{
		{
			name: "view of the target cannot be derived",
			arrange: func(t *testing.T, chain *finalityTestChain, db ethdb.Database, head *types.Header) {
				chain.dropBodies(head, 20)
			},
			wantErr: errUnknownBlock,
		},
		{
			name: "last vote cannot be read",
			arrange: func(t *testing.T, chain *finalityTestChain, db ethdb.Database, head *types.Header) {
				if err := db.Put(finalityVoteKey(finalityTestVoter), []byte("{")); err != nil {
					t.Fatalf("store unreadable vote: %v", err)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, forks := newFinalityVoteTestForks(t)
			db := rawdb.NewMemoryDatabase()
			engine := New(chain.config.Posv, db)
			chain.head = forks["main"][62].header
			tt.arrange(t, chain, db, chain.head)
			before, _ := db.Get(finalityVoteKey(finalityTestVoter))

			isAuthorized, err := engine.AuthorizeVote(chain, forks["main"][30].header, finalityTestVoter)

			if err == nil || (tt.wantErr != nil && !errors.Is(err, tt.wantErr)) {
				t.Fatalf("error: have %v, want %v", err, tt.wantErr)
			}
			if isAuthorized {
				t.Fatalf("decision on block 30: have true, want false")
			}
			after, _ := db.Get(finalityVoteKey(finalityTestVoter))
			if string(after) != string(before) {
				t.Fatalf("stored vote: have %q, want it unchanged as %q", after, before)
			}
		})
	}
}

func TestAuthorizeVoteAllowsOneOfConcurrentRequests(t *testing.T) {
	chain, forks := newFinalityVoteTestForks(t)
	engine := New(chain.config.Posv, rawdb.NewMemoryDatabase())
	chain.head = forks["main"][62].header
	target := forks["main"][30].header
	const requests = 16
	decisions := make([]bool, requests)
	failures := make([]error, requests)

	var group sync.WaitGroup
	for i := 0; i < requests; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			decisions[i], failures[i] = engine.AuthorizeVote(chain, target, finalityTestVoter)
		}(i)
	}
	group.Wait()

	authorized := 0
	for i := range decisions {
		if failures[i] != nil {
			t.Fatalf("request %d: unexpected error: %v", i, failures[i])
		}
		if decisions[i] {
			authorized++
		}
	}
	if authorized != 1 {
		t.Fatalf("authorized requests for one block: have %d, want 1", authorized)
	}
}

// newFinalityVoteTestForks returns a chain with three forks of 62 blocks, each by block number:
//   - "main": every target is justified in the block after it, so each target has the target before it as source.
//   - "quiet": leaves main after block 16 and includes no vote, so every target from 30 has block 15 as source.
//   - "bare": leaves main after block 10 and includes no vote, so every target has the root as source.
func newFinalityVoteTestForks(t *testing.T) (*finalityTestChain, map[string]map[uint64]finalityTestTip) {
	t.Helper()
	chain, quorum := newFinalityTrackerTestChain(t, 0)
	isNoTarget := func(target uint64) bool { return false }

	forks := map[string]map[uint64]finalityTestTip{"main": chain.tipsTo(t, chain.start(t), 62, quorum, isEveryTarget)}
	chain.fork = 1
	forks["quiet"] = chain.tipsTo(t, forks["main"][16], 62, quorum, isNoTarget)
	chain.fork = 2
	forks["bare"] = chain.tipsTo(t, forks["main"][10], 62, quorum, isNoTarget)
	return chain, forks
}

// requireVoteDecision fails unless the engine decides want on a vote of voter for target.
func requireVoteDecision(t *testing.T, engine *Posv, chain *finalityTestChain, target *types.Header, voter common.Address, want bool) {
	t.Helper()
	isAuthorized, err := engine.AuthorizeVote(chain, target, voter)
	if err != nil {
		t.Fatalf("authorize vote for block %d: %v", target.Number, err)
	}
	if isAuthorized != want {
		t.Fatalf("decision on block %d: have %v, want %v", target.Number, isAuthorized, want)
	}
}

// requireLastVoteTarget fails unless the last vote stored for voter has want as target; a nil want expects no stored vote.
func requireLastVoteTarget(t *testing.T, db ethdb.Database, voter common.Address, want *types.Header) {
	t.Helper()
	vote, err := loadFinalityVote(db, voter)
	switch {
	case err != nil:
		t.Fatalf("read last vote: %v", err)
	case want == nil && vote != nil:
		t.Fatalf("last vote: have block %d, want none", vote.Target.Number)
	case want != nil && vote == nil:
		t.Fatalf("last vote: have none, want block %d", want.Number)
	case want != nil && vote.Target != blockRefOf(want):
		t.Fatalf("last vote: have block %d (%s), want block %d (%s)", vote.Target.Number, vote.Target.Hash, want.Number, want.Hash())
	}
}
