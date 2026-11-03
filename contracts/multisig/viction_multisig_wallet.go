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

package Multisigwallet

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/multisig/contract"
)

type MultisigWallet struct {
	*contract.MultisigSession
	contractBackend bind.ContractBackend
}

func NewMultisigWallet(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*MultisigWallet, error) {
	blockSigner, err := contract.NewMultisig(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &MultisigWallet{
		&contract.MultisigSession{
			Contract:     blockSigner,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployMultisigWallet(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend, _owners []common.Address, _required *big.Int) (common.Address, *MultisigWallet, error) {
	blockSignerAddr, _, _, err := contract.DeployMultisig(transactOpts, contractBackend, _owners, _required)
	if err != nil {
		return blockSignerAddr, nil, err
	}

	blockSigner, err := NewMultisigWallet(transactOpts, blockSignerAddr, contractBackend)
	if err != nil {
		return blockSignerAddr, nil, err
	}

	return blockSignerAddr, blockSigner, nil
}
