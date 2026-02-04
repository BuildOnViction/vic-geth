package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (pool *TxPool) isSufficientFunds(tx *types.Transaction, from common.Address) bool {
	return pool.currentState.GetBalance(from).Cmp(tx.Cost()) < 0
}

func (pool *TxPool) validateVRC25Transaction(tx *types.Transaction, from common.Address) error {
	return nil
}

// validate customize Viction transaction (black list, special transaction) return bool to skip other validation
func (pool *TxPool) validateVictionTransaciton(tx *types.Transaction, from common.Address) (bool, error) {
	return false, nil
}
