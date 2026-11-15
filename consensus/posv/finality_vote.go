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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// finalityVotePrefix + voter address -> last vote that the validator cast from this node.
var finalityVotePrefix = []byte("posv-finality-vote-")

// finalityVote is a vote of a validator: the target it signed, and the source that target had.
type finalityVote struct {
	Target BlockRef `json:"target"`
	Source BlockRef `json:"source"` // Zero when the view of the target knew no justified block
}

// Decide whether voter may sign the block at header under the finality rule, and record the vote when it may.
// A vote is allowed when the block is on the canonical chain, is above the last target that voter signed, and has a source that is not below the source of that vote.
// A validator that follows this never signs two blocks at one height, nor a target whose source lies below one it already built on, so two conflicting blocks cannot both gather the votes that finalize them.
// The vote is stored before the decision is returned: a crash between the decision and the BlockSign transaction costs that vote, and never leads to a second one for the same height.
// A block that is not a target, as every block before the Prometheus fork, needs no permission: the decision is true and nothing is recorded.
func (c *Posv) AuthorizeVote(chain consensus.ChainReader, header *types.Header, voter common.Address) (bool, error) {
	p, err := newFinalityParams(chain.Config())
	if errors.Is(err, errFinalityInactive) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	number := header.Number.Uint64()
	if number < p.firstTarget || number%p.interval != 0 {
		return true, nil
	}

	c.voteLock.Lock()
	defer c.voteLock.Unlock()

	target := blockRefOf(header)
	if !isCanonical(chain, target) {
		log.Info("[PoSV] Vote refused: block is not on the canonical chain", "number", number, "hash", target.Hash, "voter", voter)
		return false, nil
	}
	view, err := c.finalityView(chain, header)
	if err != nil {
		return false, err
	}
	// Tally of a target opens with the block justified at the target itself, so the target's own view holds its source.
	vote := &finalityVote{Target: target, Source: view.Justified}

	last, err := loadFinalityVote(c.db, voter)
	if err != nil {
		return false, err
	}
	if last != nil && vote.Target.Number <= last.Target.Number {
		log.Info("[PoSV] Vote refused: block is not above the last signed block", "number", number, "hash", target.Hash, "lastNumber", last.Target.Number, "lastHash", last.Target.Hash, "voter", voter)
		return false, nil
	}
	if last != nil && vote.Source.Number < last.Source.Number {
		log.Info("[PoSV] Vote refused: source is below the source of the last signed block", "number", number, "hash", target.Hash, "source", vote.Source.Number, "lastSource", last.Source.Number, "voter", voter)
		return false, nil
	}
	if err := storeFinalityVote(c.db, voter, vote); err != nil {
		return false, err
	}
	return true, nil
}

// Return the database key of the last vote of voter.
func finalityVoteKey(voter common.Address) []byte {
	return append(append([]byte(nil), finalityVotePrefix...), voter.Bytes()...)
}

// Return the last vote stored for voter, or nil when it never voted from this node.
// A stored vote that cannot be read is an error, never an absent vote: without it, a second vote for one height cannot be ruled out.
func loadFinalityVote(db ethdb.KeyValueReader, voter common.Address) (*finalityVote, error) {
	key := finalityVoteKey(voter)
	if isStored, err := db.Has(key); err != nil || !isStored {
		return nil, err
	}
	blob, err := db.Get(key)
	if err != nil {
		return nil, err
	}
	vote := new(finalityVote)
	if err := json.Unmarshal(blob, vote); err != nil {
		return nil, fmt.Errorf("reading last vote of %s: %w", voter, err)
	}
	return vote, nil
}

// Store vote as the last vote of voter.
func storeFinalityVote(db ethdb.KeyValueWriter, voter common.Address, vote *finalityVote) error {
	blob, err := json.Marshal(vote)
	if err != nil {
		return err
	}
	return db.Put(finalityVoteKey(voter), blob)
}
