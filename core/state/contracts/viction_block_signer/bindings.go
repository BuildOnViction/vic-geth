// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package viction_block_signer

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

// VictionBlockSignerMetaData contains all meta data concerning the VictionBlockSigner contract.
var VictionBlockSignerMetaData = bind.MetaData{
	ABI: "[{\"constant\":false,\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\"},{\"name\":\"_blockHash\",\"type\":\"bytes32\"}],\"name\":\"sign\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_blockHash\",\"type\":\"bytes32\"}],\"name\":\"getSigners\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"epochNumber\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_epochNumber\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_signer\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_blockNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"_blockHash\",\"type\":\"bytes32\"}],\"name\":\"Sign\",\"type\":\"event\"}]",
	ID:  "VictionBlockSigner",
	Bin: "0x6060604052341561000f57600080fd5b604051602080610386833981016040528080516002555050610350806100366000396000f3006060604052600436106100565763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663e341eaa4811461005b578063e7ec6aef14610076578063f4145a83146100df575b600080fd5b341561006657600080fd5b610074600435602435610104565b005b341561008157600080fd5b61008c600435610227565b60405160208082528190810183818151815260200191508051906020019060200280838360005b838110156100cb5780820151838201526020016100b3565b505050509050019250505060405180910390f35b34156100ea57600080fd5b6100f26102ac565b60405190815260200160405180910390f35b438290101561011257600080fd5b600280546101289184910263ffffffff6102b216565b43111561013457600080fd5b600082815260016020819052604090912080549091810161015583826102c8565b5060009182526020808320919091018390558282528190526040902080546001810161018183826102c8565b506000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff19163373ffffffffffffffffffffffffffffffffffffffff8116919091179091557f62855fa22e051687c32ac285857751f6d3f2c100c72756d8d30cb7ecb1f64f5490838360405173ffffffffffffffffffffffffffffffffffffffff909316835260208301919091526040808301919091526060909101905180910390a15050565b61022f6102f1565b600082815260208181526040918290208054909290918281020190519081016040528092919081815260200182805480156102a057602002820191906000526020600020905b815473ffffffffffffffffffffffffffffffffffffffff168152600190910190602001808311610275575b50505050509050919050565b60025481565b6000828201838110156102c157fe5b9392505050565b8154818355818115116102ec576000838152602090206102ec918101908301610303565b505050565b60206040519081016040526000815290565b61032191905b8082111561031d5760008155600101610309565b5090565b905600a165627a7a72305820a8ceddaea8e4ae00991e2ae81c8c88e160dd8770f255523282c24c2df4c30ec70029",
}

// VictionBlockSigner is an auto generated Go binding around an Ethereum contract.
type VictionBlockSigner struct {
	abi abi.ABI
}

// NewVictionBlockSigner creates a new instance of VictionBlockSigner.
func NewVictionBlockSigner() *VictionBlockSigner {
	parsed, err := VictionBlockSignerMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &VictionBlockSigner{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *VictionBlockSigner) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackConstructor is the Go binding used to pack the parameters required for
// contract deployment.
//
// Solidity: constructor(uint256 _epochNumber) returns()
func (victionBlockSigner *VictionBlockSigner) PackConstructor(_epochNumber *big.Int) []byte {
	enc, err := victionBlockSigner.abi.Pack("", _epochNumber)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackEpochNumber is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xf4145a83.
//
// Solidity: function epochNumber() view returns(uint256)
func (victionBlockSigner *VictionBlockSigner) PackEpochNumber() []byte {
	enc, err := victionBlockSigner.abi.Pack("epochNumber")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackEpochNumber is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xf4145a83.
//
// Solidity: function epochNumber() view returns(uint256)
func (victionBlockSigner *VictionBlockSigner) UnpackEpochNumber(data []byte) (*big.Int, error) {
	out, err := victionBlockSigner.abi.Unpack("epochNumber", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackGetSigners is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xe7ec6aef.
//
// Solidity: function getSigners(bytes32 _blockHash) view returns(address[])
func (victionBlockSigner *VictionBlockSigner) PackGetSigners(blockHash [32]byte) []byte {
	enc, err := victionBlockSigner.abi.Pack("getSigners", blockHash)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetSigners is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xe7ec6aef.
//
// Solidity: function getSigners(bytes32 _blockHash) view returns(address[])
func (victionBlockSigner *VictionBlockSigner) UnpackGetSigners(data []byte) ([]common.Address, error) {
	out, err := victionBlockSigner.abi.Unpack("getSigners", data)
	if err != nil {
		return *new([]common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)
	return out0, err
}

// PackSign is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xe341eaa4.
//
// Solidity: function sign(uint256 _blockNumber, bytes32 _blockHash) returns()
func (victionBlockSigner *VictionBlockSigner) PackSign(blockNumber *big.Int, blockHash [32]byte) []byte {
	enc, err := victionBlockSigner.abi.Pack("sign", blockNumber, blockHash)
	if err != nil {
		panic(err)
	}
	return enc
}

// VictionBlockSignerSign represents a Sign event raised by the VictionBlockSigner contract.
type VictionBlockSignerSign struct {
	Signer      common.Address
	BlockNumber *big.Int
	BlockHash   [32]byte
	Raw         *types.Log // Blockchain specific contextual infos
}

const VictionBlockSignerSignEventName = "Sign"

// ContractEventName returns the user-defined event name.
func (VictionBlockSignerSign) ContractEventName() string {
	return VictionBlockSignerSignEventName
}

// UnpackSignEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event Sign(address _signer, uint256 _blockNumber, bytes32 _blockHash)
func (victionBlockSigner *VictionBlockSigner) UnpackSignEvent(log *types.Log) (*VictionBlockSignerSign, error) {
	event := "Sign"
	if log.Topics[0] != victionBlockSigner.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(VictionBlockSignerSign)
	if len(log.Data) > 0 {
		if err := victionBlockSigner.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range victionBlockSigner.abi.Events[event].Inputs {
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
