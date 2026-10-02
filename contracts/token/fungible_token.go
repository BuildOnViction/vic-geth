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

package token

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/token/contract"
)

type FungibleToken struct {
	*contract.FungibleTokenSession
	contractBackend bind.ContractBackend
}

func NewFungibleToken(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*FungibleToken, error) {
	fungibleToken, err := contract.NewFungibleToken(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &FungibleToken{
		&contract.FungibleTokenSession{
			Contract:     fungibleToken,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployFungibleToken(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend, name string, symbol string, decimals uint8) (common.Address, *FungibleToken, error) {
	fungibleTokenAddr, _, _, err := contract.DeployFungibleToken(transactOpts, contractBackend, name, symbol, decimals)
	if err != nil {
		return fungibleTokenAddr, nil, err
	}

	fungibleToken, err := NewFungibleToken(transactOpts, fungibleTokenAddr, contractBackend)
	if err != nil {
		return fungibleTokenAddr, nil, err
	}

	return fungibleTokenAddr, fungibleToken, nil
}
