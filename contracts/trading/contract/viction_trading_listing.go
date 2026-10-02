// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// TradingListingABI is the input ABI used to generate the binding from.
const TradingListingABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"tokens\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"}],\"name\":\"getTokenStatus\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"}],\"name\":\"apply\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"}]"

// TradingListingBin is the compiled bytecode used for deploying new contracts.
var TradingListingBin = "0x608060405234801561001057600080fd5b506102be806100206000396000f3006080604052600436106100565763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416639d63848a811461005b578063a3ff31b5146100c0578063c6b32f34146100f5575b600080fd5b34801561006757600080fd5b5061007061010b565b60408051602080825283518183015283519192839290830191858101910280838360005b838110156100ac578181015183820152602001610094565b505050509050019250505060405180910390f35b3480156100cc57600080fd5b506100e1600160a060020a036004351661016d565b604080519115158252519081900360200190f35b610109600160a060020a036004351661018b565b005b6060600080548060200260200160405190810160405280929190818152602001828054801561016357602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610145575b5050505050905090565b600160a060020a031660009081526001602052604090205460ff1690565b80600160a060020a03811615156101a157600080fd5b600160a060020a03811660009081526001602081905260409091205460ff16151514156101cd57600080fd5b683635c9adc5dea0000034146101e257600080fd5b6040516068903480156108fc02916000818181858888f1935050505015801561020f573d6000803e3d6000fd5b505060008054600180820183557f290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563909101805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a039490941693841790556040805160208082018352838252948452919093529190209051815460ff19169015151790555600a165627a7a723058206d2dc0ce827743c25efa82f99e7830ade39d28e17f4d651573f89e0460a6626a0029"

// DeployTradingListing deploys a new Ethereum contract, binding an instance of TradingListing to it.
func DeployTradingListing(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *TradingListing, error) {
	parsed, err := abi.JSON(strings.NewReader(TradingListingABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(TradingListingBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TradingListing{TradingListingCaller: TradingListingCaller{contract: contract}, TradingListingTransactor: TradingListingTransactor{contract: contract}, TradingListingFilterer: TradingListingFilterer{contract: contract}}, nil
}

// TradingListing is an auto generated Go binding around an Ethereum contract.
type TradingListing struct {
	TradingListingCaller     // Read-only binding to the contract
	TradingListingTransactor // Write-only binding to the contract
	TradingListingFilterer   // Log filterer for contract events
}

// TradingListingCaller is an auto generated read-only Go binding around an Ethereum contract.
type TradingListingCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TradingListingTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TradingListingTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TradingListingFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TradingListingFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TradingListingSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TradingListingSession struct {
	Contract     *TradingListing   // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// TradingListingCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TradingListingCallerSession struct {
	Contract *TradingListingCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts         // Call options to use throughout this session
}

// TradingListingTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TradingListingTransactorSession struct {
	Contract     *TradingListingTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts         // Transaction auth options to use throughout this session
}

// TradingListingRaw is an auto generated low-level Go binding around an Ethereum contract.
type TradingListingRaw struct {
	Contract *TradingListing // Generic contract binding to access the raw methods on
}

// TradingListingCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TradingListingCallerRaw struct {
	Contract *TradingListingCaller // Generic read-only contract binding to access the raw methods on
}

// TradingListingTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TradingListingTransactorRaw struct {
	Contract *TradingListingTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTradingListing creates a new instance of TradingListing, bound to a specific deployed contract.
func NewTradingListing(address common.Address, backend bind.ContractBackend) (*TradingListing, error) {
	contract, err := bindTradingListing(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TradingListing{TradingListingCaller: TradingListingCaller{contract: contract}, TradingListingTransactor: TradingListingTransactor{contract: contract}, TradingListingFilterer: TradingListingFilterer{contract: contract}}, nil
}

// NewTradingListingCaller creates a new read-only instance of TradingListing, bound to a specific deployed contract.
func NewTradingListingCaller(address common.Address, caller bind.ContractCaller) (*TradingListingCaller, error) {
	contract, err := bindTradingListing(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TradingListingCaller{contract: contract}, nil
}

// NewTradingListingTransactor creates a new write-only instance of TradingListing, bound to a specific deployed contract.
func NewTradingListingTransactor(address common.Address, transactor bind.ContractTransactor) (*TradingListingTransactor, error) {
	contract, err := bindTradingListing(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TradingListingTransactor{contract: contract}, nil
}

// NewTradingListingFilterer creates a new log filterer instance of TradingListing, bound to a specific deployed contract.
func NewTradingListingFilterer(address common.Address, filterer bind.ContractFilterer) (*TradingListingFilterer, error) {
	contract, err := bindTradingListing(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TradingListingFilterer{contract: contract}, nil
}

// bindTradingListing binds a generic wrapper to an already deployed contract.
func bindTradingListing(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(TradingListingABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TradingListing *TradingListingRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TradingListing.Contract.TradingListingCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TradingListing *TradingListingRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TradingListing.Contract.TradingListingTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TradingListing *TradingListingRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TradingListing.Contract.TradingListingTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TradingListing *TradingListingCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TradingListing.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TradingListing *TradingListingTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TradingListing.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TradingListing *TradingListingTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TradingListing.Contract.contract.Transact(opts, method, params...)
}

// GetTokenStatus is a free data retrieval call binding the contract method 0xa3ff31b5.
//
// Solidity: function getTokenStatus(address token) view returns(bool)
func (_TradingListing *TradingListingCaller) GetTokenStatus(opts *bind.CallOpts, token common.Address) (bool, error) {
	var out []interface{}
	err := _TradingListing.contract.Call(opts, &out, "getTokenStatus", token)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// GetTokenStatus is a free data retrieval call binding the contract method 0xa3ff31b5.
//
// Solidity: function getTokenStatus(address token) view returns(bool)
func (_TradingListing *TradingListingSession) GetTokenStatus(token common.Address) (bool, error) {
	return _TradingListing.Contract.GetTokenStatus(&_TradingListing.CallOpts, token)
}

// GetTokenStatus is a free data retrieval call binding the contract method 0xa3ff31b5.
//
// Solidity: function getTokenStatus(address token) view returns(bool)
func (_TradingListing *TradingListingCallerSession) GetTokenStatus(token common.Address) (bool, error) {
	return _TradingListing.Contract.GetTokenStatus(&_TradingListing.CallOpts, token)
}

// Tokens is a free data retrieval call binding the contract method 0x9d63848a.
//
// Solidity: function tokens() view returns(address[])
func (_TradingListing *TradingListingCaller) Tokens(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _TradingListing.contract.Call(opts, &out, "tokens")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// Tokens is a free data retrieval call binding the contract method 0x9d63848a.
//
// Solidity: function tokens() view returns(address[])
func (_TradingListing *TradingListingSession) Tokens() ([]common.Address, error) {
	return _TradingListing.Contract.Tokens(&_TradingListing.CallOpts)
}

// Tokens is a free data retrieval call binding the contract method 0x9d63848a.
//
// Solidity: function tokens() view returns(address[])
func (_TradingListing *TradingListingCallerSession) Tokens() ([]common.Address, error) {
	return _TradingListing.Contract.Tokens(&_TradingListing.CallOpts)
}

// Apply is a paid mutator transaction binding the contract method 0xc6b32f34.
//
// Solidity: function apply(address token) payable returns()
func (_TradingListing *TradingListingTransactor) Apply(opts *bind.TransactOpts, token common.Address) (*types.Transaction, error) {
	return _TradingListing.contract.Transact(opts, "apply", token)
}

// Apply is a paid mutator transaction binding the contract method 0xc6b32f34.
//
// Solidity: function apply(address token) payable returns()
func (_TradingListing *TradingListingSession) Apply(token common.Address) (*types.Transaction, error) {
	return _TradingListing.Contract.Apply(&_TradingListing.TransactOpts, token)
}

// Apply is a paid mutator transaction binding the contract method 0xc6b32f34.
//
// Solidity: function apply(address token) payable returns()
func (_TradingListing *TradingListingTransactorSession) Apply(token common.Address) (*types.Transaction, error) {
	return _TradingListing.Contract.Apply(&_TradingListing.TransactOpts, token)
}
