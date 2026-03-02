// Code generated via abigen V2 - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package viction_validator

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

// VictionValidatorMetaData contains all meta data concerning the VictionValidator contract.
var VictionValidatorMetaData = bind.MetaData{
	ABI: "[{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"propose\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"},{\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"unvote\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getCandidates\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"getWithdrawCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"getVoters\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getWithdrawBlockNumbers\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"},{\"name\":\"_voter\",\"type\":\"address\"}],\"name\":\"getVoterCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"candidates\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\"},{\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"getCandidateCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"vote\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"candidateCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"voterWithdrawDelay\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"resign\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"getCandidateOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"maxValidatorNumber\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"candidateWithdrawDelay\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"isCandidate\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"minCandidateCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"minVoterCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_candidates\",\"type\":\"address[]\"},{\"name\":\"_caps\",\"type\":\"uint256[]\"},{\"name\":\"_firstOwner\",\"type\":\"address\"},{\"name\":\"_minCandidateCap\",\"type\":\"uint256\"},{\"name\":\"_minVoterCap\",\"type\":\"uint256\"},{\"name\":\"_maxValidatorNumber\",\"type\":\"uint256\"},{\"name\":\"_candidateWithdrawDelay\",\"type\":\"uint256\"},{\"name\":\"_voterWithdrawDelay\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_voter\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Vote\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_voter\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Unvote\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Propose\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"Resign\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_blockNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Withdraw\",\"type\":\"event\"}]",
	ID:  "VictionValidator",
	Bin: "0x6060604052600060045534156200001557600080fd5b60405162001415380380620014158339810160405280805182019190602001805182019190602001805191906020018051919060200180519190602001805191906020018051919060200180516005879055600686905560078590556008849055600981905591506000905088516004555060005b88518110156200027c576003805460018101620000a883826200028b565b916000526020600020900160008b8481518110620000c257fe5b90602001906020020151909190916101000a815481600160a060020a030219169083600160a060020a031602179055505060606040519081016040908152600160a060020a03891682526001602083015281018983815181106200012257fe5b906020019060200201519052600160008b84815181106200013f57fe5b90602001906020020151600160a060020a03168152602081019190915260400160002081518154600160a060020a031916600160a060020a039190911617815560208201518154901515740100000000000000000000000000000000000000000260a060020a60ff0219909116178155604082015160019091015550600260008a8381518110620001cc57fe5b90602001906020020151600160a060020a0316815260208101919091526040016000208054600181016200020183826200028b565b50600091825260208220018054600160a060020a031916600160a060020a038a16179055600554906001908b84815181106200023957fe5b90602001906020020151600160a060020a03908116825260208083019390935260409182016000908120918c16815260029091019092529020556001016200008a565b505050505050505050620002db565b815481835581811511620002b257600083815260209020620002b2918101908301620002b7565b505050565b620002d891905b80821115620002d45760008155600101620002be565b5090565b90565b61112a80620002eb6000396000f3006060604052600436106101115763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166301267951811461011657806302aa9be21461012c57806306a49fce1461014e57806315febd68146101b45780632d15cc04146101dc5780632f9c4bba146101fb578063302b68721461020e5780633477ee2e14610233578063441a3e701461026557806358e7525f1461027e5780636dd7d8ea1461029d578063a9a981a3146102b1578063a9ff959e146102c4578063ae6e43f5146102d7578063b642facd146102f6578063d09f1ab414610315578063d161c76714610328578063d51b9e931461033b578063d55b7dff1461036e578063f8ac9dd514610381575b600080fd5b61012a600160a060020a0360043516610394565b005b341561013757600080fd5b61012a600160a060020a0360043516602435610616565b341561015957600080fd5b610161610849565b60405160208082528190810183818151815260200191508051906020019060200280838360005b838110156101a0578082015183820152602001610188565b505050509050019250505060405180910390f35b34156101bf57600080fd5b6101ca6004356108b2565b60405190815260200160405180910390f35b34156101e757600080fd5b610161600160a060020a03600435166108d6565b341561020657600080fd5b610161610963565b341561021957600080fd5b6101ca600160a060020a03600435811690602435166109e5565b341561023e57600080fd5b610249600435610a14565b604051600160a060020a03909116815260200160405180910390f35b341561027057600080fd5b61012a600435602435610a3c565b341561028957600080fd5b6101ca600160a060020a0360043516610ba3565b61012a600160a060020a0360043516610bc2565b34156102bc57600080fd5b6101ca610d7f565b34156102cf57600080fd5b6101ca610d85565b34156102e257600080fd5b61012a600160a060020a0360043516610d8b565b341561030157600080fd5b610249600160a060020a0360043516611022565b341561032057600080fd5b6101ca611040565b341561033357600080fd5b6101ca611046565b341561034657600080fd5b61035a600160a060020a036004351661104c565b604051901515815260200160405180910390f35b341561037957600080fd5b6101ca611071565b341561038c57600080fd5b6101ca611077565b6005546000903410156103a657600080fd5b600160a060020a038216600090815260016020526040902054829060a060020a900460ff16156103d557600080fd5b600160a060020a03831660009081526001602081905260409091200154610402903463ffffffff61107d16565b91506003805480600101828161041891906110a5565b506000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a03851617905560606040519081016040908152600160a060020a0333811683526001602080850182905283850187905291871660009081529152208151815473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a03919091161781556020820151815490151560a060020a0274ff0000000000000000000000000000000000000000199091161781556040820151600191820155600160a060020a03808616600090815260209283526040808220339093168252600290920190925290205461051d91503463ffffffff61107d16565b600160a060020a038085166000908152600160208181526040808420339095168452600290940190529190209190915560045461055f9163ffffffff61107d16565b600455600160a060020a038316600090815260026020526040902080546001810161058a83826110a5565b506000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff191633600160a060020a038116919091179091557f7635f1d87b47fba9f2b09e56eb4be75cca030e0cb179c1602ac9261d39a8f5c1908434604051600160a060020a039384168152919092166020820152604080820192909252606001905180910390a1505050565b600160a060020a0380831660009081526001602090815260408083203390941683526002909301905290812054839083908190101561065457600080fd5b600160a060020a03828116600090815260016020526040902054338216911614156106c257600554600160a060020a0380841660009081526001602090815260408083203390941683526002909301905220546106b7908363ffffffff61109316565b10156106c257600080fd5b600160a060020a038516600090815260016020819052604090912001546106ef908563ffffffff61109316565b600160a060020a038087166000908152600160208181526040808420928301959095553390931682526002019091522054610730908563ffffffff61109316565b600160a060020a03808716600090815260016020908152604080832033909416835260029093019052205560095461076e904363ffffffff61107d16565b600160a060020a0333166000908152602081815260408083208484529091529020549093506107a3908563ffffffff61107d16565b600160a060020a03331660008181526020818152604080832088845280835290832094909455918152905260019081018054909181016107e383826110a5565b5060009182526020909120018390557faa0e554f781c3c3b2be110a0557f260f11af9a8aa2c64bc1e7a31dbb21e32fa2338686604051600160a060020a039384168152919092166020820152604080820192909252606001905180910390a15050505050565b6108516110ce565b60038054806020026020016040519081016040528092919081815260200182805480156108a757602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610889575b505050505090505b90565b33600160a060020a0316600090815260208181526040808320938352929052205490565b6108de6110ce565b6002600083600160a060020a0316600160a060020a0316815260200190815260200160002080548060200260200160405190810160405280929190818152602001828054801561095757602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610939575b50505050509050919050565b61096b6110ce565b60008033600160a060020a0316600160a060020a031681526020019081526020016000206001018054806020026020016040519081016040528092919081815260200182805480156108a757602002820191906000526020600020905b8154815260200190600101908083116109c8575050505050905090565b600160a060020a0391821660009081526001602090815260408083209390941682526002909201909152205490565b6003805482908110610a2257fe5b600091825260209091200154600160a060020a0316905081565b60008282828211610a4c57600080fd5b4382901015610a5a57600080fd5b600160a060020a03331660009081526020818152604080832085845290915281205411610a8657600080fd5b600160a060020a0333166000908152602081905260409020600101805483919083908110610ab057fe5b60009182526020909120015414610ac657600080fd5b600160a060020a03331660008181526020818152604080832089845280835290832080549084905593835291905260010180549194509085908110610b0757fe5b6000918252602082200155600160a060020a03331683156108fc0284604051600060405180830381858888f193505050501515610b4357600080fd5b7ff279e6a1f5e320cca91135676d9cb6e44ca8a08c0b88342bcdb1144f6511b5683386856040518084600160a060020a0316600160a060020a03168152602001838152602001828152602001935050505060405180910390a15050505050565b600160a060020a03166000908152600160208190526040909120015490565b600654341015610bd157600080fd5b600160a060020a038116600090815260016020526040902054819060a060020a900460ff161515610c0157600080fd5b600160a060020a03821660009081526001602081905260409091200154610c2e903463ffffffff61107d16565b600160a060020a0380841660009081526001602081815260408084209283019590955533909316825260020190915220541515610cc057600160a060020a0382166000908152600260205260409020805460018101610c8d83826110a5565b506000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff191633600160a060020a03161790555b600160a060020a038083166000908152600160209081526040808320339094168352600290930190522054610cfb903463ffffffff61107d16565b600160a060020a03808416600090815260016020908152604080832033948516845260020190915290819020929092557f66a9138482c99e9baf08860110ef332cc0c23b4a199a53593d8db0fc8f96fbfc918490349051600160a060020a039384168152919092166020820152604080820192909252606001905180910390a15050565b60045481565b60095481565b600160a060020a038181166000908152600160205260408120549091829182918591338216911614610dbc57600080fd5b600160a060020a038516600090815260016020526040902054859060a060020a900460ff161515610dec57600080fd5b600160a060020a0386166000908152600160208190526040909120805474ff000000000000000000000000000000000000000019169055600454610e359163ffffffff61109316565b600455600094505b600354851015610ebf5785600160a060020a0316600386815481101515610e6057fe5b600091825260209091200154600160a060020a03161415610eb4576003805486908110610e8957fe5b6000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff19169055610ebf565b600190940193610e3d565b600160a060020a03808716600081815260016020818152604080842033909616845260028601825283205493909252908190529190910154909450610f0a908563ffffffff61109316565b600160a060020a0380881660009081526001602081815260408084209283019590955533909316825260020190915290812055600854610f50904363ffffffff61107d16565b600160a060020a033316600090815260208181526040808320848452909152902054909350610f85908563ffffffff61107d16565b600160a060020a0333166000818152602081815260408083208884528083529083209490945591815290526001908101805490918101610fc583826110a5565b5060009182526020909120018390557f4edf3e325d0063213a39f9085522994a1c44bea5f39e7d63ef61260a1e58c6d33387604051600160a060020a039283168152911660208201526040908101905180910390a1505050505050565b600160a060020a039081166000908152600160205260409020541690565b60075481565b60085481565b600160a060020a031660009081526001602052604090205460a060020a900460ff1690565b60055481565b60065481565b60008282018381101561108c57fe5b9392505050565b60008282111561109f57fe5b50900390565b8154818355818115116110c9576000838152602090206110c99181019083016110e0565b505050565b60206040519081016040526000815290565b6108af91905b808211156110fa57600081556001016110e6565b50905600a165627a7a72305820555de7c5131842a4fccb258fccd95ae1539019bb744b4253893b37fed1b3d8e90029",
}

// VictionValidator is an auto generated Go binding around an Ethereum contract.
type VictionValidator struct {
	abi abi.ABI
}

// NewVictionValidator creates a new instance of VictionValidator.
func NewVictionValidator() *VictionValidator {
	parsed, err := VictionValidatorMetaData.ParseABI()
	if err != nil {
		panic(errors.New("invalid ABI: " + err.Error()))
	}
	return &VictionValidator{abi: *parsed}
}

// Instance creates a wrapper for a deployed contract instance at the given address.
// Use this to create the instance object passed to abigen v2 library functions Call, Transact, etc.
func (c *VictionValidator) Instance(backend bind.ContractBackend, addr common.Address) *bind.BoundContract {
	return bind.NewBoundContract(addr, c.abi, backend, backend, backend)
}

// PackConstructor is the Go binding used to pack the parameters required for
// contract deployment.
//
// Solidity: constructor(address[] _candidates, uint256[] _caps, address _firstOwner, uint256 _minCandidateCap, uint256 _minVoterCap, uint256 _maxValidatorNumber, uint256 _candidateWithdrawDelay, uint256 _voterWithdrawDelay) returns()
func (victionValidator *VictionValidator) PackConstructor(_candidates []common.Address, _caps []*big.Int, _firstOwner common.Address, _minCandidateCap *big.Int, _minVoterCap *big.Int, _maxValidatorNumber *big.Int, _candidateWithdrawDelay *big.Int, _voterWithdrawDelay *big.Int) []byte {
	enc, err := victionValidator.abi.Pack("", _candidates, _caps, _firstOwner, _minCandidateCap, _minVoterCap, _maxValidatorNumber, _candidateWithdrawDelay, _voterWithdrawDelay)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackCandidateCount is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xa9a981a3.
//
// Solidity: function candidateCount() view returns(uint256)
func (victionValidator *VictionValidator) PackCandidateCount() []byte {
	enc, err := victionValidator.abi.Pack("candidateCount")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackCandidateCount is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xa9a981a3.
//
// Solidity: function candidateCount() view returns(uint256)
func (victionValidator *VictionValidator) UnpackCandidateCount(data []byte) (*big.Int, error) {
	out, err := victionValidator.abi.Unpack("candidateCount", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackCandidateWithdrawDelay is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xd161c767.
//
// Solidity: function candidateWithdrawDelay() view returns(uint256)
func (victionValidator *VictionValidator) PackCandidateWithdrawDelay() []byte {
	enc, err := victionValidator.abi.Pack("candidateWithdrawDelay")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackCandidateWithdrawDelay is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xd161c767.
//
// Solidity: function candidateWithdrawDelay() view returns(uint256)
func (victionValidator *VictionValidator) UnpackCandidateWithdrawDelay(data []byte) (*big.Int, error) {
	out, err := victionValidator.abi.Unpack("candidateWithdrawDelay", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackCandidates is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x3477ee2e.
//
// Solidity: function candidates(uint256 ) view returns(address)
func (victionValidator *VictionValidator) PackCandidates(arg0 *big.Int) []byte {
	enc, err := victionValidator.abi.Pack("candidates", arg0)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackCandidates is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x3477ee2e.
//
// Solidity: function candidates(uint256 ) view returns(address)
func (victionValidator *VictionValidator) UnpackCandidates(data []byte) (common.Address, error) {
	out, err := victionValidator.abi.Unpack("candidates", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, err
}

// PackGetCandidateCap is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x58e7525f.
//
// Solidity: function getCandidateCap(address _candidate) view returns(uint256)
func (victionValidator *VictionValidator) PackGetCandidateCap(candidate common.Address) []byte {
	enc, err := victionValidator.abi.Pack("getCandidateCap", candidate)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetCandidateCap is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x58e7525f.
//
// Solidity: function getCandidateCap(address _candidate) view returns(uint256)
func (victionValidator *VictionValidator) UnpackGetCandidateCap(data []byte) (*big.Int, error) {
	out, err := victionValidator.abi.Unpack("getCandidateCap", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackGetCandidateOwner is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xb642facd.
//
// Solidity: function getCandidateOwner(address _candidate) view returns(address)
func (victionValidator *VictionValidator) PackGetCandidateOwner(candidate common.Address) []byte {
	enc, err := victionValidator.abi.Pack("getCandidateOwner", candidate)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetCandidateOwner is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xb642facd.
//
// Solidity: function getCandidateOwner(address _candidate) view returns(address)
func (victionValidator *VictionValidator) UnpackGetCandidateOwner(data []byte) (common.Address, error) {
	out, err := victionValidator.abi.Unpack("getCandidateOwner", data)
	if err != nil {
		return *new(common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	return out0, err
}

// PackGetCandidates is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x06a49fce.
//
// Solidity: function getCandidates() view returns(address[])
func (victionValidator *VictionValidator) PackGetCandidates() []byte {
	enc, err := victionValidator.abi.Pack("getCandidates")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetCandidates is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x06a49fce.
//
// Solidity: function getCandidates() view returns(address[])
func (victionValidator *VictionValidator) UnpackGetCandidates(data []byte) ([]common.Address, error) {
	out, err := victionValidator.abi.Unpack("getCandidates", data)
	if err != nil {
		return *new([]common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)
	return out0, err
}

// PackGetVoterCap is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x302b6872.
//
// Solidity: function getVoterCap(address _candidate, address _voter) view returns(uint256)
func (victionValidator *VictionValidator) PackGetVoterCap(candidate common.Address, voter common.Address) []byte {
	enc, err := victionValidator.abi.Pack("getVoterCap", candidate, voter)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetVoterCap is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x302b6872.
//
// Solidity: function getVoterCap(address _candidate, address _voter) view returns(uint256)
func (victionValidator *VictionValidator) UnpackGetVoterCap(data []byte) (*big.Int, error) {
	out, err := victionValidator.abi.Unpack("getVoterCap", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackGetVoters is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2d15cc04.
//
// Solidity: function getVoters(address _candidate) view returns(address[])
func (victionValidator *VictionValidator) PackGetVoters(candidate common.Address) []byte {
	enc, err := victionValidator.abi.Pack("getVoters", candidate)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetVoters is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2d15cc04.
//
// Solidity: function getVoters(address _candidate) view returns(address[])
func (victionValidator *VictionValidator) UnpackGetVoters(data []byte) ([]common.Address, error) {
	out, err := victionValidator.abi.Unpack("getVoters", data)
	if err != nil {
		return *new([]common.Address), err
	}
	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)
	return out0, err
}

// PackGetWithdrawBlockNumbers is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x2f9c4bba.
//
// Solidity: function getWithdrawBlockNumbers() view returns(uint256[])
func (victionValidator *VictionValidator) PackGetWithdrawBlockNumbers() []byte {
	enc, err := victionValidator.abi.Pack("getWithdrawBlockNumbers")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetWithdrawBlockNumbers is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x2f9c4bba.
//
// Solidity: function getWithdrawBlockNumbers() view returns(uint256[])
func (victionValidator *VictionValidator) UnpackGetWithdrawBlockNumbers(data []byte) ([]*big.Int, error) {
	out, err := victionValidator.abi.Unpack("getWithdrawBlockNumbers", data)
	if err != nil {
		return *new([]*big.Int), err
	}
	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)
	return out0, err
}

// PackGetWithdrawCap is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x15febd68.
//
// Solidity: function getWithdrawCap(uint256 _blockNumber) view returns(uint256)
func (victionValidator *VictionValidator) PackGetWithdrawCap(blockNumber *big.Int) []byte {
	enc, err := victionValidator.abi.Pack("getWithdrawCap", blockNumber)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackGetWithdrawCap is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0x15febd68.
//
// Solidity: function getWithdrawCap(uint256 _blockNumber) view returns(uint256)
func (victionValidator *VictionValidator) UnpackGetWithdrawCap(data []byte) (*big.Int, error) {
	out, err := victionValidator.abi.Unpack("getWithdrawCap", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackIsCandidate is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xd51b9e93.
//
// Solidity: function isCandidate(address _candidate) view returns(bool)
func (victionValidator *VictionValidator) PackIsCandidate(candidate common.Address) []byte {
	enc, err := victionValidator.abi.Pack("isCandidate", candidate)
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackIsCandidate is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xd51b9e93.
//
// Solidity: function isCandidate(address _candidate) view returns(bool)
func (victionValidator *VictionValidator) UnpackIsCandidate(data []byte) (bool, error) {
	out, err := victionValidator.abi.Unpack("isCandidate", data)
	if err != nil {
		return *new(bool), err
	}
	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)
	return out0, err
}

// PackMaxValidatorNumber is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xd09f1ab4.
//
// Solidity: function maxValidatorNumber() view returns(uint256)
func (victionValidator *VictionValidator) PackMaxValidatorNumber() []byte {
	enc, err := victionValidator.abi.Pack("maxValidatorNumber")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackMaxValidatorNumber is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xd09f1ab4.
//
// Solidity: function maxValidatorNumber() view returns(uint256)
func (victionValidator *VictionValidator) UnpackMaxValidatorNumber(data []byte) (*big.Int, error) {
	out, err := victionValidator.abi.Unpack("maxValidatorNumber", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackMinCandidateCap is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xd55b7dff.
//
// Solidity: function minCandidateCap() view returns(uint256)
func (victionValidator *VictionValidator) PackMinCandidateCap() []byte {
	enc, err := victionValidator.abi.Pack("minCandidateCap")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackMinCandidateCap is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xd55b7dff.
//
// Solidity: function minCandidateCap() view returns(uint256)
func (victionValidator *VictionValidator) UnpackMinCandidateCap(data []byte) (*big.Int, error) {
	out, err := victionValidator.abi.Unpack("minCandidateCap", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackMinVoterCap is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xf8ac9dd5.
//
// Solidity: function minVoterCap() view returns(uint256)
func (victionValidator *VictionValidator) PackMinVoterCap() []byte {
	enc, err := victionValidator.abi.Pack("minVoterCap")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackMinVoterCap is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xf8ac9dd5.
//
// Solidity: function minVoterCap() view returns(uint256)
func (victionValidator *VictionValidator) UnpackMinVoterCap(data []byte) (*big.Int, error) {
	out, err := victionValidator.abi.Unpack("minVoterCap", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackPropose is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x01267951.
//
// Solidity: function propose(address _candidate) payable returns()
func (victionValidator *VictionValidator) PackPropose(candidate common.Address) []byte {
	enc, err := victionValidator.abi.Pack("propose", candidate)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackResign is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xae6e43f5.
//
// Solidity: function resign(address _candidate) returns()
func (victionValidator *VictionValidator) PackResign(candidate common.Address) []byte {
	enc, err := victionValidator.abi.Pack("resign", candidate)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackUnvote is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x02aa9be2.
//
// Solidity: function unvote(address _candidate, uint256 _cap) returns()
func (victionValidator *VictionValidator) PackUnvote(candidate common.Address, cap *big.Int) []byte {
	enc, err := victionValidator.abi.Pack("unvote", candidate, cap)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackVote is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x6dd7d8ea.
//
// Solidity: function vote(address _candidate) payable returns()
func (victionValidator *VictionValidator) PackVote(candidate common.Address) []byte {
	enc, err := victionValidator.abi.Pack("vote", candidate)
	if err != nil {
		panic(err)
	}
	return enc
}

// PackVoterWithdrawDelay is the Go binding used to pack the parameters required for calling
// the contract method with ID 0xa9ff959e.
//
// Solidity: function voterWithdrawDelay() view returns(uint256)
func (victionValidator *VictionValidator) PackVoterWithdrawDelay() []byte {
	enc, err := victionValidator.abi.Pack("voterWithdrawDelay")
	if err != nil {
		panic(err)
	}
	return enc
}

// UnpackVoterWithdrawDelay is the Go binding that unpacks the parameters returned
// from invoking the contract method with ID 0xa9ff959e.
//
// Solidity: function voterWithdrawDelay() view returns(uint256)
func (victionValidator *VictionValidator) UnpackVoterWithdrawDelay(data []byte) (*big.Int, error) {
	out, err := victionValidator.abi.Unpack("voterWithdrawDelay", data)
	if err != nil {
		return new(big.Int), err
	}
	out0 := abi.ConvertType(out[0], new(big.Int)).(*big.Int)
	return out0, err
}

// PackWithdraw is the Go binding used to pack the parameters required for calling
// the contract method with ID 0x441a3e70.
//
// Solidity: function withdraw(uint256 _blockNumber, uint256 _index) returns()
func (victionValidator *VictionValidator) PackWithdraw(blockNumber *big.Int, index *big.Int) []byte {
	enc, err := victionValidator.abi.Pack("withdraw", blockNumber, index)
	if err != nil {
		panic(err)
	}
	return enc
}

// VictionValidatorPropose represents a Propose event raised by the VictionValidator contract.
type VictionValidatorPropose struct {
	Owner     common.Address
	Candidate common.Address
	Cap       *big.Int
	Raw       *types.Log // Blockchain specific contextual infos
}

const VictionValidatorProposeEventName = "Propose"

// ContractEventName returns the user-defined event name.
func (VictionValidatorPropose) ContractEventName() string {
	return VictionValidatorProposeEventName
}

// UnpackProposeEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event Propose(address _owner, address _candidate, uint256 _cap)
func (victionValidator *VictionValidator) UnpackProposeEvent(log *types.Log) (*VictionValidatorPropose, error) {
	event := "Propose"
	if log.Topics[0] != victionValidator.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(VictionValidatorPropose)
	if len(log.Data) > 0 {
		if err := victionValidator.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range victionValidator.abi.Events[event].Inputs {
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

// VictionValidatorResign represents a Resign event raised by the VictionValidator contract.
type VictionValidatorResign struct {
	Owner     common.Address
	Candidate common.Address
	Raw       *types.Log // Blockchain specific contextual infos
}

const VictionValidatorResignEventName = "Resign"

// ContractEventName returns the user-defined event name.
func (VictionValidatorResign) ContractEventName() string {
	return VictionValidatorResignEventName
}

// UnpackResignEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event Resign(address _owner, address _candidate)
func (victionValidator *VictionValidator) UnpackResignEvent(log *types.Log) (*VictionValidatorResign, error) {
	event := "Resign"
	if log.Topics[0] != victionValidator.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(VictionValidatorResign)
	if len(log.Data) > 0 {
		if err := victionValidator.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range victionValidator.abi.Events[event].Inputs {
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

// VictionValidatorUnvote represents a Unvote event raised by the VictionValidator contract.
type VictionValidatorUnvote struct {
	Voter     common.Address
	Candidate common.Address
	Cap       *big.Int
	Raw       *types.Log // Blockchain specific contextual infos
}

const VictionValidatorUnvoteEventName = "Unvote"

// ContractEventName returns the user-defined event name.
func (VictionValidatorUnvote) ContractEventName() string {
	return VictionValidatorUnvoteEventName
}

// UnpackUnvoteEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event Unvote(address _voter, address _candidate, uint256 _cap)
func (victionValidator *VictionValidator) UnpackUnvoteEvent(log *types.Log) (*VictionValidatorUnvote, error) {
	event := "Unvote"
	if log.Topics[0] != victionValidator.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(VictionValidatorUnvote)
	if len(log.Data) > 0 {
		if err := victionValidator.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range victionValidator.abi.Events[event].Inputs {
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

// VictionValidatorVote represents a Vote event raised by the VictionValidator contract.
type VictionValidatorVote struct {
	Voter     common.Address
	Candidate common.Address
	Cap       *big.Int
	Raw       *types.Log // Blockchain specific contextual infos
}

const VictionValidatorVoteEventName = "Vote"

// ContractEventName returns the user-defined event name.
func (VictionValidatorVote) ContractEventName() string {
	return VictionValidatorVoteEventName
}

// UnpackVoteEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event Vote(address _voter, address _candidate, uint256 _cap)
func (victionValidator *VictionValidator) UnpackVoteEvent(log *types.Log) (*VictionValidatorVote, error) {
	event := "Vote"
	if log.Topics[0] != victionValidator.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(VictionValidatorVote)
	if len(log.Data) > 0 {
		if err := victionValidator.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range victionValidator.abi.Events[event].Inputs {
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

// VictionValidatorWithdraw represents a Withdraw event raised by the VictionValidator contract.
type VictionValidatorWithdraw struct {
	Owner       common.Address
	BlockNumber *big.Int
	Cap         *big.Int
	Raw         *types.Log // Blockchain specific contextual infos
}

const VictionValidatorWithdrawEventName = "Withdraw"

// ContractEventName returns the user-defined event name.
func (VictionValidatorWithdraw) ContractEventName() string {
	return VictionValidatorWithdrawEventName
}

// UnpackWithdrawEvent is the Go binding that unpacks the event data emitted
// by contract.
//
// Solidity: event Withdraw(address _owner, uint256 _blockNumber, uint256 _cap)
func (victionValidator *VictionValidator) UnpackWithdrawEvent(log *types.Log) (*VictionValidatorWithdraw, error) {
	event := "Withdraw"
	if log.Topics[0] != victionValidator.abi.Events[event].ID {
		return nil, errors.New("event signature mismatch")
	}
	out := new(VictionValidatorWithdraw)
	if len(log.Data) > 0 {
		if err := victionValidator.abi.UnpackIntoInterface(out, event, log.Data); err != nil {
			return nil, err
		}
	}
	var indexed abi.Arguments
	for _, arg := range victionValidator.abi.Events[event].Inputs {
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
