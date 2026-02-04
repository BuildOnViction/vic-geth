package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

func (pool *TxPool) isSufficientFunds(tx *types.Transaction, from common.Address) bool {
	return pool.currentState.GetBalance(from).Cmp(tx.Cost()) < 0
}

// validate customize Viction transaction (black list, special transaction) return bool to skip other validation
func (pool *TxPool) validateSpecialTransaction(tx *types.Transaction, from common.Address) (bool, error) {
	if err := state.ValidateVRC25Transaction(pool.currentState, pool.chainconfig.VRC25Contract, from, *tx.To(), tx.Data()); err != nil {
		return false, err
	}
	if pool.isSufficientFunds(tx, from) {
		return true, nil
	}
	return false, nil
}
