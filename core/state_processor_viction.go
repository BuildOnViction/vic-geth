// Copyright 2026 The Vic-geth Authors
// This file provides vic-extensions to the geth.
package core

import (
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

type VictionProcessorState struct {
}


func (p *StateProcessor) BeforeProcess(block *types.Block, statedb *state.StateDB) error {
	p.victionState = &VictionProcessorState{}
	return nil
}

func (p *StateProcessor) AfterProcess(block *types.Block, statedb *state.StateDB) error {
	return nil
}

func (p *StateProcessor) BeforeApplyTransaction(tx *types.Transaction, msg Message, statedb *state.StateDB) error {
	return nil
}

func (p *StateProcessor) AfterApplyTransaction(tx *types.Transaction, msg Message, statedb *state.StateDB, receipt *types.Receipt, usedGas uint64, err error) error {
	return nil
}
