// Copyright (c) 2026 Viction
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

package blocksigner

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/blocksigner/contract"
	"github.com/ethereum/go-ethereum/core/types"
)

// [7s62] MergeSignRange controls sign-tx submission frequency after TIP2019.
// Before TIP2019 every imported block triggers a sign tx; after TIP2019 only
// every MergeSignRange-th block does, reducing pool spam (mirrors victionchain).
const MergeSignRange = uint64(15)

// [7s62] CreateTxSign builds an unsigned BlockSigner.sign(blockNumber, blockHash)
// transaction.  The calldata is ABI-encoded manually to avoid an ethclient
// round-trip; the caller is responsible for signing and injecting into the pool.
//
//	selector: sign(uint256,bytes32) → e341eaa4
func CreateTxSign(blockNumber *big.Int, blockHash common.Hash, nonce uint64, blockSignerAddr common.Address) *types.Transaction {
	data := common.Hex2Bytes("e341eaa4")
	inputData := append(data, common.LeftPadBytes(blockNumber.Bytes(), 32)...)
	inputData = append(inputData, blockHash.Bytes()...)
	return types.NewTransaction(nonce, blockSignerAddr, big.NewInt(0), 200000, big.NewInt(0), inputData)
}

type BlockSigner struct {
	*contract.BlockSignerSession
	contractBackend bind.ContractBackend
}

func NewBlockSigner(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*BlockSigner, error) {
	blockSigner, err := contract.NewBlockSigner(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &BlockSigner{
		&contract.BlockSignerSession{
			Contract:     blockSigner,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployBlockSigner(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend, epochNumber *big.Int) (common.Address, *BlockSigner, error) {
	blockSignerAddr, _, _, err := contract.DeployBlockSigner(transactOpts, contractBackend, epochNumber)
	if err != nil {
		return blockSignerAddr, nil, err
	}

	blockSigner, err := NewBlockSigner(transactOpts, blockSignerAddr, contractBackend)
	if err != nil {
		return blockSignerAddr, nil, err
	}

	return blockSignerAddr, blockSigner, nil
}
