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

package zerogas

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/zerogas/contract"
	"math/big"
)

type ZeroGas struct {
	*contract.ZeroGasSession
	contractBackend bind.ContractBackend
}

func NewZeroGas(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*ZeroGas, error) {
	contractObject, err := contract.NewZeroGas(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &ZeroGas{
		&contract.ZeroGasSession{
			Contract:     contractObject,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployZeroGas(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend, minApply *big.Int) (common.Address, *ZeroGas, error) {
	contractAddr, _, _, err := contract.DeployZeroGas(transactOpts, contractBackend, minApply)
	if err != nil {
		return contractAddr, nil, err
	}
	contractObject, err := NewZeroGas(transactOpts, contractAddr, contractBackend)
	if err != nil {
		return contractAddr, nil, err
	}

	return contractAddr, contractObject, nil
}
