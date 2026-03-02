// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package viction_randomize

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

// VictionRandomizeMetaData contains all meta data concerning the VictionRandomize contract.
var VictionRandomizeMetaData = bind.MetaData{
	ABI: "[{\"constant\":true,\"inputs\":[{\"name\":\"_validator\",\"type\":\"address\"}],\"name\":\"getSecret\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_secret\",\"type\":\"bytes32[]\"}],\"name\":\"setSecret\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_validator\",\"type\":\"address\"}],\"name\":\"getOpening\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_opening\",\"type\":\"bytes32\"}],\"name\":\"setOpening\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]",
	ID:  "VictionRandomize",
	Bin: "0x6060604052341561000f57600080fd5b6103368061001e6000396000f3006060604052600436106100615763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663284180fc811461006657806334d38600146100d8578063d442d6cc14610129578063e11f5ba21461015a575b600080fd5b341561007157600080fd5b610085600160a060020a0360043516610170565b60405160208082528190810183818151815260200191508051906020019060200280838360005b838110156100c45780820151838201526020016100ac565b505050509050019250505060405180910390f35b34156100e357600080fd5b61012760046024813581810190830135806020818102016040519081016040528093929190818152602001838360200280828437509496506101f395505050505050565b005b341561013457600080fd5b610148600160a060020a0360043516610243565b60405190815260200160405180910390f35b341561016557600080fd5b61012760043561025e565b61017861028e565b60008083600160a060020a0316600160a060020a031681526020019081526020016000208054806020026020016040519081016040528092919081815260200182805480156101e757602002820191906000526020600020905b815481526001909101906020018083116101d2575b50505050509050919050565b610384430661032081101561020757600080fd5b610352811061021557600080fd5b600160a060020a033316600090815260208190526040902082805161023e9291602001906102a0565b505050565b600160a060020a031660009081526001602052604090205490565b610384430661035281101561027257600080fd5b50600160a060020a033316600090815260016020526040902055565b60206040519081016040526000815290565b8280548282559060005260206000209081019282156102dd579160200282015b828111156102dd57825182556020909201916001909101906102c0565b506102e99291506102ed565b5090565b61030791905b808211156102e957600081556001016102f3565b905600a165627a7a7230582034991c8dc4001fc254f3ba2811c05d2e7d29bee3908946ca56d1545b2c852de20029",
}

// VictionRandomize is an auto generated Go binding around an Ethereum contract.
type VictionRandomize struct {
	abi abi.ABI
}

// NewVictionRandomize creates a new instance of VictionRandomize.
func NewVictionRandomize() *VictionRandomize {
	parsed, err := VictionRandomizeMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &VictionRandomize{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *VictionRandomize) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackGetOpening is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xd442d6cc.
//
// Solidity: function getOpening(address _validator) view returns(bytes32)
func (victionRandomize *VictionRandomize) PackGetOpening(validator common.Address) []byte {
	enc, err := victionRandomize.abi.Pack("getOpening", validator)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetOpening is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xd442d6cc.
//
// Solidity: function getOpening(address _validator) view returns(bytes32)
func (victionRandomize *VictionRandomize) UnpackGetOpening(data []byte) ([32]byte, error) {
	out, err := victionRandomize.abi.Unpack("getOpening", data)
	if err != nil {
		return *new([32]byte), err
	}
	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)
	return out0, err
}

// PackGetSecret is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x284180fc.
//
// Solidity: function getSecret(address _validator) view returns(bytes32[])
func (victionRandomize *VictionRandomize) PackGetSecret(validator common.Address) []byte {
	enc, err := victionRandomize.abi.Pack("getSecret", validator)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetSecret is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x284180fc.
//
// Solidity: function getSecret(address _validator) view returns(bytes32[])
func (victionRandomize *VictionRandomize) UnpackGetSecret(data []byte) ([][32]byte, error) {
	out, err := victionRandomize.abi.Unpack("getSecret", data)
	if err != nil {
		return *new([][32]byte), err
	}
	out0 := *abi.ConvertType(out[0], new([][32]byte)).(*[][32]byte)
	return out0, err
}

// PackSetOpening is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xe11f5ba2.
//
// Solidity: function setOpening(bytes32 _opening) returns()
func (victionRandomize *VictionRandomize) PackSetOpening(opening [32]byte) []byte {
	enc, err := victionRandomize.abi.Pack("setOpening", opening)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackSetSecret is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x34d38600.
//
// Solidity: function setSecret(bytes32[] _secret) returns()
func (victionRandomize *VictionRandomize) PackSetSecret(secret [][32]byte) []byte {
	enc, err := victionRandomize.abi.Pack("setSecret", secret)
	if err != nil {
		panic(err)
	}
	return enc
}
