// Copyright 2014 The go-ethereum Authors
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
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/sortlgc"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/log"
)

func (bc *BlockChain) UpdateValidators() error {
	engine, ok := bc.Engine().(*posv.Posv)
	if bc.Config().Posv == nil || !ok {
		return ErrPosvRequired
	}
	log.Info("[Blockchain] Preparing new validators list for next epoch.")

	contractAddress := bc.chainConfig.Viction.ValidatorContract
	if contractAddress == (common.Address{}) {
		return ErrNoValidatorContract
	}

	var candidates []common.Address
	stateDB, err := bc.State()
	if err != nil {
		return fmt.Errorf("failed to get state at block #%v: %v", bc.CurrentHeader().Number, err)
	}
	candidates = stateDB.VicGetCandidates(contractAddress)

	var validators []posv.ValidatorInfo
	for _, candidate := range candidates {
		_, cap := stateDB.VicGetValidatorInfo(contractAddress, candidate)
		if candidate.String() != "0x0000000000000000000000000000000000000000" {
			validators = append(validators, posv.ValidatorInfo{Address: candidate, Capacity: cap})
		}
	}
	if len(validators) == 0 {
		return ErrNoValidators
	}

	header := bc.CurrentHeader()
	if bc.Config().IsAtlas(header.Number) {
		sort.SliceStable(validators, func(i, j int) bool {
			return validators[i].Capacity.Cmp(validators[j].Capacity) >= 0
		})
	} else {
		sortlgc.Slice(validators, func(i, j int) bool {
			return validators[i].Capacity.Cmp(validators[j].Capacity) >= 0
		})
	}

	vs := make([]common.Address, 0)
	if len(validators) > int(bc.chainConfig.Viction.ValidatorMaxCount) {
		for _, v := range validators[:bc.chainConfig.Viction.ValidatorMaxCount] {
			vs = append(vs, v.Address)
		}
	} else {
		for _, v := range validators {
			vs = append(vs, v.Address)
		}
	}
	err = engine.SetCheckpointSigners(bc, header, vs)
	if err != nil {
		return err
	}

	log.Info("[Blockchain] Updated validators list for next epoch", "count", len(vs))
	return nil
}
