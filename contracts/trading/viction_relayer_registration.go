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

package trading

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/trading/contract"
)

type RelayerRegistration struct {
	*contract.RelayerRegistrationSession
	contractBackend bind.ContractBackend
}

func NewRelayerRegistration(transactOpts *bind.TransactOpts, contractAddr common.Address, contractBackend bind.ContractBackend) (*RelayerRegistration, error) {
	relayerRegistration, err := contract.NewRelayerRegistration(contractAddr, contractBackend)
	if err != nil {
		return nil, err
	}

	return &RelayerRegistration{
		&contract.RelayerRegistrationSession{
			Contract:     relayerRegistration,
			TransactOpts: *transactOpts,
		},
		contractBackend,
	}, nil
}

func DeployRelayerRegistration(transactOpts *bind.TransactOpts, contractBackend bind.ContractBackend, tradingListing common.Address, maxRelayers *big.Int, maxTokenList *big.Int, minDeposit *big.Int) (common.Address, *RelayerRegistration, error) {
	relayerRegistrationAddr, _, _, err := contract.DeployRelayerRegistration(transactOpts, contractBackend, tradingListing, maxRelayers, maxTokenList, minDeposit)
	if err != nil {
		return relayerRegistrationAddr, nil, err
	}

	relayerRegistration, err := NewRelayerRegistration(transactOpts, relayerRegistrationAddr, contractBackend)
	if err != nil {
		return relayerRegistrationAddr, nil, err
	}

	return relayerRegistrationAddr, relayerRegistration, nil
}
