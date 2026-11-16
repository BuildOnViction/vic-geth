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

import "github.com/ethereum/go-ethereum/common"

// FinalityThresholdPercent is the share of an epoch's validators, in percent, that must sign a block to finalize it.
const FinalityThresholdPercent = 75

// FinalityTally counts the validators that signed one block toward its finality.
// The zero value is not usable; create it with NewFinalityTally.
type FinalityTally struct {
	Number     uint64 // number of the signed block
	validators map[common.Address]struct{}
	signers    map[common.Address]struct{}
}

// Create a tally for the block at number that counts only signers among validators.
func NewFinalityTally(number uint64, validators []common.Address) *FinalityTally {
	set := make(map[common.Address]struct{}, len(validators))
	for _, validator := range validators {
		set[validator] = struct{}{}
	}
	return &FinalityTally{
		Number:     number,
		validators: set,
		signers:    make(map[common.Address]struct{}, len(validators)),
	}
}

// Record addr as a signer of the block; an address outside the validator set or already recorded changes nothing.
func (t *FinalityTally) AddSigner(addr common.Address) {
	if _, ok := t.validators[addr]; ok {
		t.signers[addr] = struct{}{}
	}
}

// Report whether the signers cover FinalityThresholdPercent of the validators; an empty validator set never does.
func (t *FinalityTally) HasQuorum() bool {
	if len(t.validators) == 0 {
		return false
	}
	return 100*len(t.signers) >= FinalityThresholdPercent*len(t.validators)
}
