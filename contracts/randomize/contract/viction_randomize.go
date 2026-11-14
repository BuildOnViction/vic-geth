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

// RandomizeABI is the input ABI used to generate the binding from.
const RandomizeABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"_validator\",\"type\":\"address\"}],\"name\":\"getSecret\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_secret\",\"type\":\"bytes32[]\"}],\"name\":\"setSecret\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_validator\",\"type\":\"address\"}],\"name\":\"getOpening\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_opening\",\"type\":\"bytes32\"}],\"name\":\"setOpening\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]"

// RandomizeBin is the compiled bytecode used for deploying new contracts.
var RandomizeBin = "0x6060604052341561000f57600080fd5b6103368061001e6000396000f3006060604052600436106100615763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663284180fc811461006657806334d38600146100d8578063d442d6cc14610129578063e11f5ba21461015a575b600080fd5b341561007157600080fd5b610085600160a060020a0360043516610170565b60405160208082528190810183818151815260200191508051906020019060200280838360005b838110156100c45780820151838201526020016100ac565b505050509050019250505060405180910390f35b34156100e357600080fd5b61012760046024813581810190830135806020818102016040519081016040528093929190818152602001838360200280828437509496506101f395505050505050565b005b341561013457600080fd5b610148600160a060020a0360043516610243565b60405190815260200160405180910390f35b341561016557600080fd5b61012760043561025e565b61017861028e565b60008083600160a060020a0316600160a060020a031681526020019081526020016000208054806020026020016040519081016040528092919081815260200182805480156101e757602002820191906000526020600020905b815481526001909101906020018083116101d2575b50505050509050919050565b610384430661032081101561020757600080fd5b610352811061021557600080fd5b600160a060020a033316600090815260208190526040902082805161023e9291602001906102a0565b505050565b600160a060020a031660009081526001602052604090205490565b610384430661035281101561027257600080fd5b50600160a060020a033316600090815260016020526040902055565b60206040519081016040526000815290565b8280548282559060005260206000209081019282156102dd579160200282015b828111156102dd57825182556020909201916001909101906102c0565b506102e99291506102ed565b5090565b61030791905b808211156102e957600081556001016102f3565b905600a165627a7a7230582034991c8dc4001fc254f3ba2811c05d2e7d29bee3908946ca56d1545b2c852de20029"

// DeployRandomize deploys a new Ethereum contract, binding an instance of Randomize to it.
func DeployRandomize(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Randomize, error) {
	parsed, err := abi.JSON(strings.NewReader(RandomizeABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(RandomizeBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Randomize{RandomizeCaller: RandomizeCaller{contract: contract}, RandomizeTransactor: RandomizeTransactor{contract: contract}, RandomizeFilterer: RandomizeFilterer{contract: contract}}, nil
}

// Randomize is an auto generated Go binding around an Ethereum contract.
type Randomize struct {
	RandomizeCaller     // Read-only binding to the contract
	RandomizeTransactor // Write-only binding to the contract
	RandomizeFilterer   // Log filterer for contract events
}

// RandomizeCaller is an auto generated read-only Go binding around an Ethereum contract.
type RandomizeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RandomizeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RandomizeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RandomizeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RandomizeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RandomizeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RandomizeSession struct {
	Contract     *Randomize        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RandomizeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RandomizeCallerSession struct {
	Contract *RandomizeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// RandomizeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RandomizeTransactorSession struct {
	Contract     *RandomizeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// RandomizeRaw is an auto generated low-level Go binding around an Ethereum contract.
type RandomizeRaw struct {
	Contract *Randomize // Generic contract binding to access the raw methods on
}

// RandomizeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RandomizeCallerRaw struct {
	Contract *RandomizeCaller // Generic read-only contract binding to access the raw methods on
}

// RandomizeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RandomizeTransactorRaw struct {
	Contract *RandomizeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRandomize creates a new instance of Randomize, bound to a specific deployed contract.
func NewRandomize(address common.Address, backend bind.ContractBackend) (*Randomize, error) {
	contract, err := bindRandomize(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Randomize{RandomizeCaller: RandomizeCaller{contract: contract}, RandomizeTransactor: RandomizeTransactor{contract: contract}, RandomizeFilterer: RandomizeFilterer{contract: contract}}, nil
}

// NewRandomizeCaller creates a new read-only instance of Randomize, bound to a specific deployed contract.
func NewRandomizeCaller(address common.Address, caller bind.ContractCaller) (*RandomizeCaller, error) {
	contract, err := bindRandomize(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RandomizeCaller{contract: contract}, nil
}

// NewRandomizeTransactor creates a new write-only instance of Randomize, bound to a specific deployed contract.
func NewRandomizeTransactor(address common.Address, transactor bind.ContractTransactor) (*RandomizeTransactor, error) {
	contract, err := bindRandomize(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RandomizeTransactor{contract: contract}, nil
}

// NewRandomizeFilterer creates a new log filterer instance of Randomize, bound to a specific deployed contract.
func NewRandomizeFilterer(address common.Address, filterer bind.ContractFilterer) (*RandomizeFilterer, error) {
	contract, err := bindRandomize(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RandomizeFilterer{contract: contract}, nil
}

// bindRandomize binds a generic wrapper to an already deployed contract.
func bindRandomize(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(RandomizeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Randomize *RandomizeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Randomize.Contract.RandomizeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Randomize *RandomizeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Randomize.Contract.RandomizeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Randomize *RandomizeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Randomize.Contract.RandomizeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Randomize *RandomizeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Randomize.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Randomize *RandomizeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Randomize.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Randomize *RandomizeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Randomize.Contract.contract.Transact(opts, method, params...)
}

// GetOpening is a free data retrieval call binding the contract method 0xd442d6cc.
//
// Solidity: function getOpening(address _validator) view returns(bytes32)
func (_Randomize *RandomizeCaller) GetOpening(opts *bind.CallOpts, _validator common.Address) ([32]byte, error) {
	var out []interface{}
	err := _Randomize.contract.Call(opts, &out, "getOpening", _validator)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetOpening is a free data retrieval call binding the contract method 0xd442d6cc.
//
// Solidity: function getOpening(address _validator) view returns(bytes32)
func (_Randomize *RandomizeSession) GetOpening(_validator common.Address) ([32]byte, error) {
	return _Randomize.Contract.GetOpening(&_Randomize.CallOpts, _validator)
}

// GetOpening is a free data retrieval call binding the contract method 0xd442d6cc.
//
// Solidity: function getOpening(address _validator) view returns(bytes32)
func (_Randomize *RandomizeCallerSession) GetOpening(_validator common.Address) ([32]byte, error) {
	return _Randomize.Contract.GetOpening(&_Randomize.CallOpts, _validator)
}

// GetSecret is a free data retrieval call binding the contract method 0x284180fc.
//
// Solidity: function getSecret(address _validator) view returns(bytes32[])
func (_Randomize *RandomizeCaller) GetSecret(opts *bind.CallOpts, _validator common.Address) ([][32]byte, error) {
	var out []interface{}
	err := _Randomize.contract.Call(opts, &out, "getSecret", _validator)

	if err != nil {
		return *new([][32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)

	return out0, err

}

// GetSecret is a free data retrieval call binding the contract method 0x284180fc.
//
// Solidity: function getSecret(address _validator) view returns(bytes32[])
func (_Randomize *RandomizeSession) GetSecret(_validator common.Address) ([][32]byte, error) {
	return _Randomize.Contract.GetSecret(&_Randomize.CallOpts, _validator)
}

// GetSecret is a free data retrieval call binding the contract method 0x284180fc.
//
// Solidity: function getSecret(address _validator) view returns(bytes32[])
func (_Randomize *RandomizeCallerSession) GetSecret(_validator common.Address) ([][32]byte, error) {
	return _Randomize.Contract.GetSecret(&_Randomize.CallOpts, _validator)
}

// SetOpening is a paid mutator transaction binding the contract method 0xe11f5ba2.
//
// Solidity: function setOpening(bytes32 _opening) returns()
func (_Randomize *RandomizeTransactor) SetOpening(opts *bind.TransactOpts, _opening [32]byte) (*types.Transaction, error) {
	return _Randomize.contract.Transact(opts, "setOpening", _opening)
}

// SetOpening is a paid mutator transaction binding the contract method 0xe11f5ba2.
//
// Solidity: function setOpening(bytes32 _opening) returns()
func (_Randomize *RandomizeSession) SetOpening(_opening [32]byte) (*types.Transaction, error) {
	return _Randomize.Contract.SetOpening(&_Randomize.TransactOpts, _opening)
}

// SetOpening is a paid mutator transaction binding the contract method 0xe11f5ba2.
//
// Solidity: function setOpening(bytes32 _opening) returns()
func (_Randomize *RandomizeTransactorSession) SetOpening(_opening [32]byte) (*types.Transaction, error) {
	return _Randomize.Contract.SetOpening(&_Randomize.TransactOpts, _opening)
}

// SetSecret is a paid mutator transaction binding the contract method 0x34d38600.
//
// Solidity: function setSecret(bytes32[] _secret) returns()
func (_Randomize *RandomizeTransactor) SetSecret(opts *bind.TransactOpts, _secret [][32]byte) (*types.Transaction, error) {
	return _Randomize.contract.Transact(opts, "setSecret", _secret)
}

// SetSecret is a paid mutator transaction binding the contract method 0x34d38600.
//
// Solidity: function setSecret(bytes32[] _secret) returns()
func (_Randomize *RandomizeSession) SetSecret(_secret [][32]byte) (*types.Transaction, error) {
	return _Randomize.Contract.SetSecret(&_Randomize.TransactOpts, _secret)
}

// SetSecret is a paid mutator transaction binding the contract method 0x34d38600.
//
// Solidity: function setSecret(bytes32[] _secret) returns()
func (_Randomize *RandomizeTransactorSession) SetSecret(_secret [][32]byte) (*types.Transaction, error) {
	return _Randomize.Contract.SetSecret(&_Randomize.TransactOpts, _secret)
}
