// Copyright (c) 2018 Tomochain
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package posv

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	lru "github.com/hashicorp/golang-lru"
)

// Vote represents a single vote that an authorized signer made to modify the
// list of authorizations.
//type Vote struct {
//	Signer    common.Address `json:"signer"`    // Authorized signer that cast this vote
//	Block     uint64         `json:"block"`     // Block number the vote was cast in (expire old votes)
//	Address   common.Address `json:"address"`   // Account being voted on to change its authorization
//	Authorize bool           `json:"authorize"` // Whether to authorize or deauthorize the voted account
//}

// Tally is a simple vote tally to keep the current score of votes. Votes that
// go against the proposal aren't counted since it's equivalent to not voting.
//type Tally struct {
//	Authorize bool `json:"authorize"` // Whether the vote is about authorizing or kicking someone
//	Votes     int  `json:"votes"`     // Number of votes until now wanting to pass the proposal
//}

// Snapshot is the state of the authorization voting at a given point in time.
type Snapshot struct {
	config   *params.PosvConfig // Consensus engine parameters to fine tune behavior
	sigcache *lru.ARCCache      // Cache of recent block signatures to speed up ecrecover

	Number  uint64                          `json:"number"`  // Block number where the snapshot was created
	Hash    common.Hash                     `json:"hash"`    // Block hash where the snapshot was created
	Signers map[common.Address]struct{}     `json:"signers"` // Set of authorized signers at this moment
	Recents map[uint64]common.Address       `json:"recents"` // Set of recent signers for spam protections
	Votes   []*clique.Vote                  `json:"votes"`   // List of votes cast in chronological order
	Tally   map[common.Address]clique.Tally `json:"tally"`   // Current vote tally to avoid recalculating
}

// [TO-DO]
// newSnapshot creates a new snapshot with the specified startup parameters. This
// method does not initialize the set of recent signers, so only ever use if for
// the genesis block.
func newSnapshot(config *params.PosvConfig, sigcache *lru.ARCCache, number uint64, hash common.Hash, signers []common.Address) *Snapshot {
	return nil
}

// [TO-DO]
// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(config *params.PosvConfig, sigcache *lru.ARCCache, db ethdb.Database, hash common.Hash) (*Snapshot, error) {
	return nil, nil
}

// [TO-DO]
// store inserts the snapshot into the database.
func (s *Snapshot) store(db ethdb.Database) error {
	return nil
}

// [TO-DO]
// copy creates a deep copy of the snapshot, though not the individual votes.
func (s *Snapshot) copy() *Snapshot {
	return nil
}

// [TO-DO]
// validVote returns whether it makes sense to cast the specified vote in the
// given snapshot context (e.g. don't try to add an already authorized signer).
func (s *Snapshot) validVote(address common.Address, authorize bool) bool {
	return false
}

// [TO-DO]
// cast adds a new vote into the tally.
func (s *Snapshot) cast(address common.Address, authorize bool) bool {
	return false
}

// [TO-DO]
// uncast removes a previously cast vote from the tally.
func (s *Snapshot) uncast(address common.Address, authorize bool) bool {
	return false
}

// [TO-DO]
// apply creates a new authorization snapshot by applying the given headers to
// the original one.
func (s *Snapshot) apply(headers []*types.Header) (*Snapshot, error) {
	return nil, nil
}

// [TO-DO]
// signers retrieves the list of authorized signers in ascending order.
func (s *Snapshot) GetSigners() []common.Address {
	return nil
}
