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
