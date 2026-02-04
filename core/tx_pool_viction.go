package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

// validate sufficient balance for transaction execution, considering VRC25 fee cap if applicable
func (pool *TxPool) validateSufficientTransaction(tx *types.Transaction, from common.Address) error {
	balance := pool.currentState.GetBalance(from)
	requiredBalance := tx.Cost()

	if tx.To() != nil {
		feeCap := state.GetFeeCapacity(pool.currentState, pool.chainconfig.VRC25Contract, from)
		if feeCap != nil {
			// VRC25 transaction
			if err := state.ValidateVRC25Transaction(pool.currentState, pool.chainconfig.VRC25Contract, from, *tx.To(), tx.Data()); err != nil {
				return err
			}

			requiredFee := new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), pool.chainconfig.VRC25GasPrice)
			if feeCap.Cmp(requiredFee) >= 0 {
				// if fee capacity is sufficient, reduce the required balance by gas fee
				requiredBalance = tx.Value()
			}
		}
	}

	if balance.Cmp(requiredBalance) < 0 {
		return ErrInsufficientFunds
	}

	return nil
}
