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

// ZeroGasABI is the input ABI used to generate the binding from.
const ZeroGasABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"minCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"}],\"name\":\"getTokenCapacity\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"tokens\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"}],\"name\":\"apply\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"}],\"name\":\"charge\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"issuer\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Apply\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"supporter\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Charge\",\"type\":\"event\"}]"

// ZeroGasBin is the compiled bytecode used for deploying new contracts.
var ZeroGasBin = "0x608060405234801561001057600080fd5b5060405160208061047d8339810160405251600055610449806100346000396000f30060806040526004361061006c5763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416633fa615b081146100715780638f3a981c146100985780639d63848a146100b9578063c6b32f341461011e578063fc6bd76a14610134575b600080fd5b34801561007d57600080fd5b50610086610148565b60408051918252519081900360200190f35b3480156100a457600080fd5b50610086600160a060020a036004351661014e565b3480156100c557600080fd5b506100ce610169565b60408051602080825283518183015283519192839290830191858101910280838360005b8381101561010a5781810151838201526020016100f2565b505050509050019250505060405180910390f35b610132600160a060020a03600435166101cb565b005b610132600160a060020a036004351661035d565b60005490565b600160a060020a031660009081526002602052604090205490565b606060018054806020026020016040519081016040528092919081815260200182805480156101c157602002820191906000526020600020905b8154600160a060020a031681526001909101906020018083116101a3575b5050505050905090565b600081600160a060020a03811615156101e357600080fd5b6000543410156101f257600080fd5b82915033600160a060020a031682600160a060020a0316631d1438486040518163ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401602060405180830381600087803b15801561025657600080fd5b505af115801561026a573d6000803e3d6000fd5b505050506040513d602081101561028057600080fd5b5051600160a060020a03161461029557600080fd5b600180548082019091557fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf601805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0385169081179091556000908152600260205260409020546103039034610404565b600160a060020a0384166000818152600260209081526040918290209390935580513481529051919233927f2d485624158277d5113a56388c3abf5c20e3511dd112123ba376d16b21e4d7169281900390910190a3505050565b80600160a060020a038116151561037357600080fd5b60005434101561038257600080fd5b600160a060020a0382166000908152600260205260409020546103ab903463ffffffff61040416565b600160a060020a0383166000818152600260209081526040918290209390935580513481529051919233927f5cffac866325fd9b2a8ea8df2f110a0058313b279402d15ae28dd324a2282e069281900390910190a35050565b60008282018381101561041657600080fd5b93925050505600a165627a7a72305820ff8b5d768ed70aba2a462acd944dd172bd1771ddf32b0dda96f2a0db1b67307f0029"

// DeployZeroGas deploys a new Ethereum contract, binding an instance of ZeroGas to it.
func DeployZeroGas(auth *bind.TransactOpts, backend bind.ContractBackend, value *big.Int) (common.Address, *types.Transaction, *ZeroGas, error) {
	parsed, err := abi.JSON(strings.NewReader(ZeroGasABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ZeroGasBin), backend, value)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ZeroGas{ZeroGasCaller: ZeroGasCaller{contract: contract}, ZeroGasTransactor: ZeroGasTransactor{contract: contract}, ZeroGasFilterer: ZeroGasFilterer{contract: contract}}, nil
}

// ZeroGas is an auto generated Go binding around an Ethereum contract.
type ZeroGas struct {
	ZeroGasCaller     // Read-only binding to the contract
	ZeroGasTransactor // Write-only binding to the contract
	ZeroGasFilterer   // Log filterer for contract events
}

// ZeroGasCaller is an auto generated read-only Go binding around an Ethereum contract.
type ZeroGasCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ZeroGasTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ZeroGasTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ZeroGasFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ZeroGasFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ZeroGasSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ZeroGasSession struct {
	Contract     *ZeroGas          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ZeroGasCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ZeroGasCallerSession struct {
	Contract *ZeroGasCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// ZeroGasTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ZeroGasTransactorSession struct {
	Contract     *ZeroGasTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// ZeroGasRaw is an auto generated low-level Go binding around an Ethereum contract.
type ZeroGasRaw struct {
	Contract *ZeroGas // Generic contract binding to access the raw methods on
}

// ZeroGasCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ZeroGasCallerRaw struct {
	Contract *ZeroGasCaller // Generic read-only contract binding to access the raw methods on
}

// ZeroGasTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ZeroGasTransactorRaw struct {
	Contract *ZeroGasTransactor // Generic write-only contract binding to access the raw methods on
}

// NewZeroGas creates a new instance of ZeroGas, bound to a specific deployed contract.
func NewZeroGas(address common.Address, backend bind.ContractBackend) (*ZeroGas, error) {
	contract, err := bindZeroGas(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ZeroGas{ZeroGasCaller: ZeroGasCaller{contract: contract}, ZeroGasTransactor: ZeroGasTransactor{contract: contract}, ZeroGasFilterer: ZeroGasFilterer{contract: contract}}, nil
}

// NewZeroGasCaller creates a new read-only instance of ZeroGas, bound to a specific deployed contract.
func NewZeroGasCaller(address common.Address, caller bind.ContractCaller) (*ZeroGasCaller, error) {
	contract, err := bindZeroGas(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ZeroGasCaller{contract: contract}, nil
}

// NewZeroGasTransactor creates a new write-only instance of ZeroGas, bound to a specific deployed contract.
func NewZeroGasTransactor(address common.Address, transactor bind.ContractTransactor) (*ZeroGasTransactor, error) {
	contract, err := bindZeroGas(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ZeroGasTransactor{contract: contract}, nil
}

// NewZeroGasFilterer creates a new log filterer instance of ZeroGas, bound to a specific deployed contract.
func NewZeroGasFilterer(address common.Address, filterer bind.ContractFilterer) (*ZeroGasFilterer, error) {
	contract, err := bindZeroGas(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ZeroGasFilterer{contract: contract}, nil
}

// bindZeroGas binds a generic wrapper to an already deployed contract.
func bindZeroGas(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ZeroGasABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ZeroGas *ZeroGasRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ZeroGas.Contract.ZeroGasCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ZeroGas *ZeroGasRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ZeroGas.Contract.ZeroGasTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ZeroGas *ZeroGasRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ZeroGas.Contract.ZeroGasTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ZeroGas *ZeroGasCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ZeroGas.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ZeroGas *ZeroGasTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ZeroGas.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ZeroGas *ZeroGasTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ZeroGas.Contract.contract.Transact(opts, method, params...)
}

// GetTokenCapacity is a free data retrieval call binding the contract method 0x8f3a981c.
//
// Solidity: function getTokenCapacity(address token) view returns(uint256)
func (_ZeroGas *ZeroGasCaller) GetTokenCapacity(opts *bind.CallOpts, token common.Address) (*big.Int, error) {
	var out []interface{}
	err := _ZeroGas.contract.Call(opts, &out, "getTokenCapacity", token)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetTokenCapacity is a free data retrieval call binding the contract method 0x8f3a981c.
//
// Solidity: function getTokenCapacity(address token) view returns(uint256)
func (_ZeroGas *ZeroGasSession) GetTokenCapacity(token common.Address) (*big.Int, error) {
	return _ZeroGas.Contract.GetTokenCapacity(&_ZeroGas.CallOpts, token)
}

// GetTokenCapacity is a free data retrieval call binding the contract method 0x8f3a981c.
//
// Solidity: function getTokenCapacity(address token) view returns(uint256)
func (_ZeroGas *ZeroGasCallerSession) GetTokenCapacity(token common.Address) (*big.Int, error) {
	return _ZeroGas.Contract.GetTokenCapacity(&_ZeroGas.CallOpts, token)
}

// MinCap is a free data retrieval call binding the contract method 0x3fa615b0.
//
// Solidity: function minCap() view returns(uint256)
func (_ZeroGas *ZeroGasCaller) MinCap(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ZeroGas.contract.Call(opts, &out, "minCap")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinCap is a free data retrieval call binding the contract method 0x3fa615b0.
//
// Solidity: function minCap() view returns(uint256)
func (_ZeroGas *ZeroGasSession) MinCap() (*big.Int, error) {
	return _ZeroGas.Contract.MinCap(&_ZeroGas.CallOpts)
}

// MinCap is a free data retrieval call binding the contract method 0x3fa615b0.
//
// Solidity: function minCap() view returns(uint256)
func (_ZeroGas *ZeroGasCallerSession) MinCap() (*big.Int, error) {
	return _ZeroGas.Contract.MinCap(&_ZeroGas.CallOpts)
}

// Tokens is a free data retrieval call binding the contract method 0x9d63848a.
//
// Solidity: function tokens() view returns(address[])
func (_ZeroGas *ZeroGasCaller) Tokens(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _ZeroGas.contract.Call(opts, &out, "tokens")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// Tokens is a free data retrieval call binding the contract method 0x9d63848a.
//
// Solidity: function tokens() view returns(address[])
func (_ZeroGas *ZeroGasSession) Tokens() ([]common.Address, error) {
	return _ZeroGas.Contract.Tokens(&_ZeroGas.CallOpts)
}

// Tokens is a free data retrieval call binding the contract method 0x9d63848a.
//
// Solidity: function tokens() view returns(address[])
func (_ZeroGas *ZeroGasCallerSession) Tokens() ([]common.Address, error) {
	return _ZeroGas.Contract.Tokens(&_ZeroGas.CallOpts)
}

// Apply is a paid mutator transaction binding the contract method 0xc6b32f34.
//
// Solidity: function apply(address token) payable returns()
func (_ZeroGas *ZeroGasTransactor) Apply(opts *bind.TransactOpts, token common.Address) (*types.Transaction, error) {
	return _ZeroGas.contract.Transact(opts, "apply", token)
}

// Apply is a paid mutator transaction binding the contract method 0xc6b32f34.
//
// Solidity: function apply(address token) payable returns()
func (_ZeroGas *ZeroGasSession) Apply(token common.Address) (*types.Transaction, error) {
	return _ZeroGas.Contract.Apply(&_ZeroGas.TransactOpts, token)
}

// Apply is a paid mutator transaction binding the contract method 0xc6b32f34.
//
// Solidity: function apply(address token) payable returns()
func (_ZeroGas *ZeroGasTransactorSession) Apply(token common.Address) (*types.Transaction, error) {
	return _ZeroGas.Contract.Apply(&_ZeroGas.TransactOpts, token)
}

// Charge is a paid mutator transaction binding the contract method 0xfc6bd76a.
//
// Solidity: function charge(address token) payable returns()
func (_ZeroGas *ZeroGasTransactor) Charge(opts *bind.TransactOpts, token common.Address) (*types.Transaction, error) {
	return _ZeroGas.contract.Transact(opts, "charge", token)
}

// Charge is a paid mutator transaction binding the contract method 0xfc6bd76a.
//
// Solidity: function charge(address token) payable returns()
func (_ZeroGas *ZeroGasSession) Charge(token common.Address) (*types.Transaction, error) {
	return _ZeroGas.Contract.Charge(&_ZeroGas.TransactOpts, token)
}

// Charge is a paid mutator transaction binding the contract method 0xfc6bd76a.
//
// Solidity: function charge(address token) payable returns()
func (_ZeroGas *ZeroGasTransactorSession) Charge(token common.Address) (*types.Transaction, error) {
	return _ZeroGas.Contract.Charge(&_ZeroGas.TransactOpts, token)
}

// ZeroGasApplyIterator is returned from FilterApply and is used to iterate over the raw logs and unpacked data for Apply events raised by the ZeroGas contract.
type ZeroGasApplyIterator struct {
	Event *ZeroGasApply // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ZeroGasApplyIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ZeroGasApply)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ZeroGasApply)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ZeroGasApplyIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ZeroGasApplyIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ZeroGasApply represents a Apply event raised by the ZeroGas contract.
type ZeroGasApply struct {
	Issuer common.Address
	Token  common.Address
	Value  *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterApply is a free log retrieval operation binding the contract event 0x2d485624158277d5113a56388c3abf5c20e3511dd112123ba376d16b21e4d716.
//
// Solidity: event Apply(address indexed issuer, address indexed token, uint256 value)
func (_ZeroGas *ZeroGasFilterer) FilterApply(opts *bind.FilterOpts, issuer []common.Address, token []common.Address) (*ZeroGasApplyIterator, error) {

	var issuerRule []interface{}
	for _, issuerItem := range issuer {
		issuerRule = append(issuerRule, issuerItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _ZeroGas.contract.FilterLogs(opts, "Apply", issuerRule, tokenRule)
	if err != nil {
		return nil, err
	}
	return &ZeroGasApplyIterator{contract: _ZeroGas.contract, event: "Apply", logs: logs, sub: sub}, nil
}

// WatchApply is a free log subscription operation binding the contract event 0x2d485624158277d5113a56388c3abf5c20e3511dd112123ba376d16b21e4d716.
//
// Solidity: event Apply(address indexed issuer, address indexed token, uint256 value)
func (_ZeroGas *ZeroGasFilterer) WatchApply(opts *bind.WatchOpts, sink chan<- *ZeroGasApply, issuer []common.Address, token []common.Address) (event.Subscription, error) {

	var issuerRule []interface{}
	for _, issuerItem := range issuer {
		issuerRule = append(issuerRule, issuerItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _ZeroGas.contract.WatchLogs(opts, "Apply", issuerRule, tokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ZeroGasApply)
				if err := _ZeroGas.contract.UnpackLog(event, "Apply", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseApply is a log parse operation binding the contract event 0x2d485624158277d5113a56388c3abf5c20e3511dd112123ba376d16b21e4d716.
//
// Solidity: event Apply(address indexed issuer, address indexed token, uint256 value)
func (_ZeroGas *ZeroGasFilterer) ParseApply(log types.Log) (*ZeroGasApply, error) {
	event := new(ZeroGasApply)
	if err := _ZeroGas.contract.UnpackLog(event, "Apply", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ZeroGasChargeIterator is returned from FilterCharge and is used to iterate over the raw logs and unpacked data for Charge events raised by the ZeroGas contract.
type ZeroGasChargeIterator struct {
	Event *ZeroGasCharge // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ZeroGasChargeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ZeroGasCharge)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ZeroGasCharge)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ZeroGasChargeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ZeroGasChargeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ZeroGasCharge represents a Charge event raised by the ZeroGas contract.
type ZeroGasCharge struct {
	Supporter common.Address
	Token     common.Address
	Value     *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterCharge is a free log retrieval operation binding the contract event 0x5cffac866325fd9b2a8ea8df2f110a0058313b279402d15ae28dd324a2282e06.
//
// Solidity: event Charge(address indexed supporter, address indexed token, uint256 value)
func (_ZeroGas *ZeroGasFilterer) FilterCharge(opts *bind.FilterOpts, supporter []common.Address, token []common.Address) (*ZeroGasChargeIterator, error) {

	var supporterRule []interface{}
	for _, supporterItem := range supporter {
		supporterRule = append(supporterRule, supporterItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _ZeroGas.contract.FilterLogs(opts, "Charge", supporterRule, tokenRule)
	if err != nil {
		return nil, err
	}
	return &ZeroGasChargeIterator{contract: _ZeroGas.contract, event: "Charge", logs: logs, sub: sub}, nil
}

// WatchCharge is a free log subscription operation binding the contract event 0x5cffac866325fd9b2a8ea8df2f110a0058313b279402d15ae28dd324a2282e06.
//
// Solidity: event Charge(address indexed supporter, address indexed token, uint256 value)
func (_ZeroGas *ZeroGasFilterer) WatchCharge(opts *bind.WatchOpts, sink chan<- *ZeroGasCharge, supporter []common.Address, token []common.Address) (event.Subscription, error) {

	var supporterRule []interface{}
	for _, supporterItem := range supporter {
		supporterRule = append(supporterRule, supporterItem)
	}
	var tokenRule []interface{}
	for _, tokenItem := range token {
		tokenRule = append(tokenRule, tokenItem)
	}

	logs, sub, err := _ZeroGas.contract.WatchLogs(opts, "Charge", supporterRule, tokenRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ZeroGasCharge)
				if err := _ZeroGas.contract.UnpackLog(event, "Charge", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseCharge is a log parse operation binding the contract event 0x5cffac866325fd9b2a8ea8df2f110a0058313b279402d15ae28dd324a2282e06.
//
// Solidity: event Charge(address indexed supporter, address indexed token, uint256 value)
func (_ZeroGas *ZeroGasFilterer) ParseCharge(log types.Log) (*ZeroGasCharge, error) {
	event := new(ZeroGasCharge)
	if err := _ZeroGas.contract.UnpackLog(event, "Charge", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
