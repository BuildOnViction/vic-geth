// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package viction_zero_gas

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = bytes.Equal
	_ = errors.New
	_ = big.NewInt
	_ = common.Big1
	_ = types.BloomLookup
	_ = abi.ConvertType
)

// VictionZeroGasMetaData contains all meta data concerning the VictionZeroGas contract.
var VictionZeroGasMetaData = bind.MetaData{
	ABI: "[{\"constant\":true,\"inputs\":[],\"name\":\"minCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"}],\"name\":\"getTokenCapacity\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"tokens\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"}],\"name\":\"apply\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"token\",\"type\":\"address\"}],\"name\":\"charge\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"value\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"issuer\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Apply\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"supporter\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"token\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Charge\",\"type\":\"event\"}]",
	ID:  "VictionZeroGas",
	Bin: "0x608060405234801561001057600080fd5b5060405160208061047d8339810160405251600055610449806100346000396000f30060806040526004361061006c5763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416633fa615b081146100715780638f3a981c146100985780639d63848a146100b9578063c6b32f341461011e578063fc6bd76a14610134575b600080fd5b34801561007d57600080fd5b50610086610148565b60408051918252519081900360200190f35b3480156100a457600080fd5b50610086600160a060020a036004351661014e565b3480156100c557600080fd5b506100ce610169565b60408051602080825283518183015283519192839290830191858101910280838360005b8381101561010a5781810151838201526020016100f2565b505050509050019250505060405180910390f35b610132600160a060020a03600435166101cb565b005b610132600160a060020a036004351661035d565b60005490565b600160a060020a031660009081526002602052604090205490565b606060018054806020026020016040519081016040528092919081815260200182805480156101c157602002820191906000526020600020905b8154600160a060020a031681526001909101906020018083116101a3575b5050505050905090565b600081600160a060020a03811615156101e357600080fd5b6000543410156101f257600080fd5b82915033600160a060020a031682600160a060020a0316631d1438486040518163ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401602060405180830381600087803b15801561025657600080fd5b505af115801561026a573d6000803e3d6000fd5b505050506040513d602081101561028057600080fd5b5051600160a060020a03161461029557600080fd5b600180548082019091557fb10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0cf601805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0385169081179091556000908152600260205260409020546103039034610404565b600160a060020a0384166000818152600260209081526040918290209390935580513481529051919233927f2d485624158277d5113a56388c3abf5c20e3511dd112123ba376d16b21e4d7169281900390910190a3505050565b80600160a060020a038116151561037357600080fd5b60005434101561038257600080fd5b600160a060020a0382166000908152600260205260409020546103ab903463ffffffff61040416565b600160a060020a0383166000818152600260209081526040918290209390935580513481529051919233927f5cffac866325fd9b2a8ea8df2f110a0058313b279402d15ae28dd324a2282e069281900390910190a35050565b60008282018381101561041657600080fd5b93925050505600a165627a7a72305820ff8b5d768ed70aba2a462acd944dd172bd1771ddf32b0dda96f2a0db1b67307f0029",
}

// VictionZeroGas is an auto generated Go binding around an Ethereum contract.
type VictionZeroGas struct {
	abi abi.ABI
}

// NewVictionZeroGas creates a new instance of VictionZeroGas.
func NewVictionZeroGas() *VictionZeroGas {
	parsed, err := VictionZeroGasMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &VictionZeroGas{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *VictionZeroGas) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackConstructor is the Go binding used to pack the parameters required for
// contract deployment.
//
// Solidity: constructor(uint256 value) returns()
func (victionZeroGas *VictionZeroGas) PackConstructor(value *big.Int) []byte {
	enc, err := victionZeroGas.abi.Pack("", value)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackApply is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xc6b32f34.
//
// Solidity: function apply(address token) payable returns()
func (victionZeroGas *VictionZeroGas) PackApply(token common.Address) []byte {
	enc, err := victionZeroGas.abi.Pack("apply", token)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackCharge is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xfc6bd76a.
//
// Solidity: function charge(address token) payable returns()
func (victionZeroGas *VictionZeroGas) PackCharge(token common.Address) []byte {
	enc, err := victionZeroGas.abi.Pack("charge", token)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackGetTokenCapacity is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x8f3a981c.
//
// Solidity: function getTokenCapacity(address token) view returns(uint256)
func (victionZeroGas *VictionZeroGas) PackGetTokenCapacity(token common.Address) []byte {
	enc, err := victionZeroGas.abi.Pack("getTokenCapacity", token)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetTokenCapacity is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x8f3a981c.
//
// Solidity: function getTokenCapacity(address token) view returns(uint256)
func (victionZeroGas *VictionZeroGas) UnpackGetTokenCapacity(data []byte) (*big.Int, error) {
	out, err := victionZeroGas.abi.Unpack("getTokenCapacity", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackMinCap is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x3fa615b0.
//
// Solidity: function minCap() view returns(uint256)
func (victionZeroGas *VictionZeroGas) PackMinCap() []byte {
	enc, err := victionZeroGas.abi.Pack("minCap")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackMinCap is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x3fa615b0.
//
// Solidity: function minCap() view returns(uint256)
func (victionZeroGas *VictionZeroGas) UnpackMinCap(data []byte) (*big.Int, error) {
	out, err := victionZeroGas.abi.Unpack("minCap", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackTokens is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x9d63848a.
//
// Solidity: function tokens() view returns(address[])
func (victionZeroGas *VictionZeroGas) PackTokens() []byte {
	enc, err := victionZeroGas.abi.Pack("tokens")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackTokens is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x9d63848a.
//
// Solidity: function tokens() view returns(address[])
func (victionZeroGas *VictionZeroGas) UnpackTokens(data []byte) ([]common.Address, error) {
	out, err := victionZeroGas.abi.Unpack("tokens", data)
	if err != nil {
		return *new([]common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)
	return out0, err
}

// VictionZeroGasApply represents a Apply event raised by the VictionZeroGas contract.
type VictionZeroGasApply struct {
	Issuer common.Address
	Token  common.Address
	Value  *big.Int
	Raw    *types.Log // Blockchain specific contextual infos
}

const VictionZeroGasApplyEventName = "Apply"

// ContractEventName returns the user-defined event name.
func (VictionZeroGasApply) ContractEventName() string {
	return VictionZeroGasApplyEventName
}

// UnpackApplyEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event Apply(address indexed issuer, address indexed token, uint256 value)
func (victionZeroGas *VictionZeroGas) UnpackApplyEvent(log *types.Log) (*VictionZeroGasApply, error) {
	event := "Apply"
	if log.Topics[0] != victionZeroGas.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(VictionZeroGasApply)
	if len(log.Data) > 0 {
		if err := victionZeroGas.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range victionZeroGas.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}

// VictionZeroGasCharge represents a Charge event raised by the VictionZeroGas contract.
type VictionZeroGasCharge struct {
	Supporter common.Address
	Token     common.Address
	Value     *big.Int
	Raw       *types.Log // Blockchain specific contextual infos
}

const VictionZeroGasChargeEventName = "Charge"

// ContractEventName returns the user-defined event name.
func (VictionZeroGasCharge) ContractEventName() string {
	return VictionZeroGasChargeEventName
}

// UnpackChargeEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event Charge(address indexed supporter, address indexed token, uint256 value)
func (victionZeroGas *VictionZeroGas) UnpackChargeEvent(log *types.Log) (*VictionZeroGasCharge, error) {
	event := "Charge"
	if log.Topics[0] != victionZeroGas.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(VictionZeroGasCharge)
	if len(log.Data) > 0 {
		if err := victionZeroGas.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range victionZeroGas.abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	if err := abi.ParseTopics(out, indexed, log.Topics[1:]); err != nil {
		return nil, err
	}
	out.Raw = log
	return out, nil
}
