package viction

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func NewSignBlockTransaction(blockNumber *big.Int, blockHash common.Hash, nonce uint64, blockSignerAddr common.Address) *types.Transaction {
	data := common.Hex2Bytes("e341eaa4")
	inputData := append(data, common.LeftPadBytes(blockNumber.Bytes(), 32)...)
	inputData = append(inputData, blockHash.Bytes()...)
	return types.NewTransaction(nonce, blockSignerAddr, big.NewInt(0), 200000, big.NewInt(0), inputData)
}
