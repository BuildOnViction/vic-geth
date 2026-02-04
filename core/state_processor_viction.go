// Copyright 2026 The Vic-geth Authors
// This file provides vic-extensions to the geth.
package core

import (
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

type victionProcessorState struct {
}


func (p *StateProcessor) beforeProcess(block *types.Block, statedb *state.StateDB) error {
	// clear previous state and initialize new
	p.victionState = &victionProcessorState{}
	return nil
}

func (p *StateProcessor) afterProcess(block *types.Block, statedb *state.StateDB) error {
	return nil
}

func (p *StateProcessor) beforeApplyTransaction(tx *types.Transaction, msg Message, statedb *state.StateDB) error {
	return nil
}

func (p *StateProcessor) afterApplyTransaction(tx *types.Transaction, msg Message, statedb *state.StateDB, receipt *types.Receipt, usedGas uint64, err error) error {
	return nil
}
