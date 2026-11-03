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

// ValidatorABI is the input ABI used to generate the binding from.
const ValidatorABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"propose\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"},{\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"unvote\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getCandidates\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\"}],\"name\":\"getWithdrawCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"getVoters\",\"outputs\":[{\"name\":\"\",\"type\":\"address[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"getWithdrawBlockNumbers\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256[]\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"},{\"name\":\"_voter\",\"type\":\"address\"}],\"name\":\"getVoterCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"candidates\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_blockNumber\",\"type\":\"uint256\"},{\"name\":\"_index\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"getCandidateCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"vote\",\"outputs\":[],\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"candidateCount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"voterWithdrawDelay\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"resign\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"getCandidateOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"maxValidatorNumber\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"candidateWithdrawDelay\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"isCandidate\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"minCandidateCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"minVoterCap\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_candidates\",\"type\":\"address[]\"},{\"name\":\"_caps\",\"type\":\"uint256[]\"},{\"name\":\"_firstOwner\",\"type\":\"address\"},{\"name\":\"_minCandidateCap\",\"type\":\"uint256\"},{\"name\":\"_minVoterCap\",\"type\":\"uint256\"},{\"name\":\"_maxValidatorNumber\",\"type\":\"uint256\"},{\"name\":\"_candidateWithdrawDelay\",\"type\":\"uint256\"},{\"name\":\"_voterWithdrawDelay\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_voter\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Vote\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_voter\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Unvote\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Propose\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_candidate\",\"type\":\"address\"}],\"name\":\"Resign\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"_owner\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"_blockNumber\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"_cap\",\"type\":\"uint256\"}],\"name\":\"Withdraw\",\"type\":\"event\"}]"

// ValidatorBin is the compiled bytecode used for deploying new contracts.
var ValidatorBin = "0x6060604052600060045534156200001557600080fd5b60405162001415380380620014158339810160405280805182019190602001805182019190602001805191906020018051919060200180519190602001805191906020018051919060200180516005879055600686905560078590556008849055600981905591506000905088516004555060005b88518110156200027c576003805460018101620000a883826200028b565b916000526020600020900160008b8481518110620000c257fe5b90602001906020020151909190916101000a815481600160a060020a030219169083600160a060020a031602179055505060606040519081016040908152600160a060020a03891682526001602083015281018983815181106200012257fe5b906020019060200201519052600160008b84815181106200013f57fe5b90602001906020020151600160a060020a03168152602081019190915260400160002081518154600160a060020a031916600160a060020a039190911617815560208201518154901515740100000000000000000000000000000000000000000260a060020a60ff0219909116178155604082015160019091015550600260008a8381518110620001cc57fe5b90602001906020020151600160a060020a0316815260208101919091526040016000208054600181016200020183826200028b565b50600091825260208220018054600160a060020a031916600160a060020a038a16179055600554906001908b84815181106200023957fe5b90602001906020020151600160a060020a03908116825260208083019390935260409182016000908120918c16815260029091019092529020556001016200008a565b505050505050505050620002db565b815481835581811511620002b257600083815260209020620002b2918101908301620002b7565b505050565b620002d891905b80821115620002d45760008155600101620002be565b5090565b90565b61112a80620002eb6000396000f3006060604052600436106101115763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166301267951811461011657806302aa9be21461012c57806306a49fce1461014e57806315febd68146101b45780632d15cc04146101dc5780632f9c4bba146101fb578063302b68721461020e5780633477ee2e14610233578063441a3e701461026557806358e7525f1461027e5780636dd7d8ea1461029d578063a9a981a3146102b1578063a9ff959e146102c4578063ae6e43f5146102d7578063b642facd146102f6578063d09f1ab414610315578063d161c76714610328578063d51b9e931461033b578063d55b7dff1461036e578063f8ac9dd514610381575b600080fd5b61012a600160a060020a0360043516610394565b005b341561013757600080fd5b61012a600160a060020a0360043516602435610616565b341561015957600080fd5b610161610849565b60405160208082528190810183818151815260200191508051906020019060200280838360005b838110156101a0578082015183820152602001610188565b505050509050019250505060405180910390f35b34156101bf57600080fd5b6101ca6004356108b2565b60405190815260200160405180910390f35b34156101e757600080fd5b610161600160a060020a03600435166108d6565b341561020657600080fd5b610161610963565b341561021957600080fd5b6101ca600160a060020a03600435811690602435166109e5565b341561023e57600080fd5b610249600435610a14565b604051600160a060020a03909116815260200160405180910390f35b341561027057600080fd5b61012a600435602435610a3c565b341561028957600080fd5b6101ca600160a060020a0360043516610ba3565b61012a600160a060020a0360043516610bc2565b34156102bc57600080fd5b6101ca610d7f565b34156102cf57600080fd5b6101ca610d85565b34156102e257600080fd5b61012a600160a060020a0360043516610d8b565b341561030157600080fd5b610249600160a060020a0360043516611022565b341561032057600080fd5b6101ca611040565b341561033357600080fd5b6101ca611046565b341561034657600080fd5b61035a600160a060020a036004351661104c565b604051901515815260200160405180910390f35b341561037957600080fd5b6101ca611071565b341561038c57600080fd5b6101ca611077565b6005546000903410156103a657600080fd5b600160a060020a038216600090815260016020526040902054829060a060020a900460ff16156103d557600080fd5b600160a060020a03831660009081526001602081905260409091200154610402903463ffffffff61107d16565b91506003805480600101828161041891906110a5565b506000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a03851617905560606040519081016040908152600160a060020a0333811683526001602080850182905283850187905291871660009081529152208151815473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a03919091161781556020820151815490151560a060020a0274ff0000000000000000000000000000000000000000199091161781556040820151600191820155600160a060020a03808616600090815260209283526040808220339093168252600290920190925290205461051d91503463ffffffff61107d16565b600160a060020a038085166000908152600160208181526040808420339095168452600290940190529190209190915560045461055f9163ffffffff61107d16565b600455600160a060020a038316600090815260026020526040902080546001810161058a83826110a5565b506000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff191633600160a060020a038116919091179091557f7635f1d87b47fba9f2b09e56eb4be75cca030e0cb179c1602ac9261d39a8f5c1908434604051600160a060020a039384168152919092166020820152604080820192909252606001905180910390a1505050565b600160a060020a0380831660009081526001602090815260408083203390941683526002909301905290812054839083908190101561065457600080fd5b600160a060020a03828116600090815260016020526040902054338216911614156106c257600554600160a060020a0380841660009081526001602090815260408083203390941683526002909301905220546106b7908363ffffffff61109316565b10156106c257600080fd5b600160a060020a038516600090815260016020819052604090912001546106ef908563ffffffff61109316565b600160a060020a038087166000908152600160208181526040808420928301959095553390931682526002019091522054610730908563ffffffff61109316565b600160a060020a03808716600090815260016020908152604080832033909416835260029093019052205560095461076e904363ffffffff61107d16565b600160a060020a0333166000908152602081815260408083208484529091529020549093506107a3908563ffffffff61107d16565b600160a060020a03331660008181526020818152604080832088845280835290832094909455918152905260019081018054909181016107e383826110a5565b5060009182526020909120018390557faa0e554f781c3c3b2be110a0557f260f11af9a8aa2c64bc1e7a31dbb21e32fa2338686604051600160a060020a039384168152919092166020820152604080820192909252606001905180910390a15050505050565b6108516110ce565b60038054806020026020016040519081016040528092919081815260200182805480156108a757602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610889575b505050505090505b90565b33600160a060020a0316600090815260208181526040808320938352929052205490565b6108de6110ce565b6002600083600160a060020a0316600160a060020a0316815260200190815260200160002080548060200260200160405190810160405280929190818152602001828054801561095757602002820191906000526020600020905b8154600160a060020a03168152600190910190602001808311610939575b50505050509050919050565b61096b6110ce565b60008033600160a060020a0316600160a060020a031681526020019081526020016000206001018054806020026020016040519081016040528092919081815260200182805480156108a757602002820191906000526020600020905b8154815260200190600101908083116109c8575050505050905090565b600160a060020a0391821660009081526001602090815260408083209390941682526002909201909152205490565b6003805482908110610a2257fe5b600091825260209091200154600160a060020a0316905081565b60008282828211610a4c57600080fd5b4382901015610a5a57600080fd5b600160a060020a03331660009081526020818152604080832085845290915281205411610a8657600080fd5b600160a060020a0333166000908152602081905260409020600101805483919083908110610ab057fe5b60009182526020909120015414610ac657600080fd5b600160a060020a03331660008181526020818152604080832089845280835290832080549084905593835291905260010180549194509085908110610b0757fe5b6000918252602082200155600160a060020a03331683156108fc0284604051600060405180830381858888f193505050501515610b4357600080fd5b7ff279e6a1f5e320cca91135676d9cb6e44ca8a08c0b88342bcdb1144f6511b5683386856040518084600160a060020a0316600160a060020a03168152602001838152602001828152602001935050505060405180910390a15050505050565b600160a060020a03166000908152600160208190526040909120015490565b600654341015610bd157600080fd5b600160a060020a038116600090815260016020526040902054819060a060020a900460ff161515610c0157600080fd5b600160a060020a03821660009081526001602081905260409091200154610c2e903463ffffffff61107d16565b600160a060020a0380841660009081526001602081815260408084209283019590955533909316825260020190915220541515610cc057600160a060020a0382166000908152600260205260409020805460018101610c8d83826110a5565b506000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff191633600160a060020a03161790555b600160a060020a038083166000908152600160209081526040808320339094168352600290930190522054610cfb903463ffffffff61107d16565b600160a060020a03808416600090815260016020908152604080832033948516845260020190915290819020929092557f66a9138482c99e9baf08860110ef332cc0c23b4a199a53593d8db0fc8f96fbfc918490349051600160a060020a039384168152919092166020820152604080820192909252606001905180910390a15050565b60045481565b60095481565b600160a060020a038181166000908152600160205260408120549091829182918591338216911614610dbc57600080fd5b600160a060020a038516600090815260016020526040902054859060a060020a900460ff161515610dec57600080fd5b600160a060020a0386166000908152600160208190526040909120805474ff000000000000000000000000000000000000000019169055600454610e359163ffffffff61109316565b600455600094505b600354851015610ebf5785600160a060020a0316600386815481101515610e6057fe5b600091825260209091200154600160a060020a03161415610eb4576003805486908110610e8957fe5b6000918252602090912001805473ffffffffffffffffffffffffffffffffffffffff19169055610ebf565b600190940193610e3d565b600160a060020a03808716600081815260016020818152604080842033909616845260028601825283205493909252908190529190910154909450610f0a908563ffffffff61109316565b600160a060020a0380881660009081526001602081815260408084209283019590955533909316825260020190915290812055600854610f50904363ffffffff61107d16565b600160a060020a033316600090815260208181526040808320848452909152902054909350610f85908563ffffffff61107d16565b600160a060020a0333166000818152602081815260408083208884528083529083209490945591815290526001908101805490918101610fc583826110a5565b5060009182526020909120018390557f4edf3e325d0063213a39f9085522994a1c44bea5f39e7d63ef61260a1e58c6d33387604051600160a060020a039283168152911660208201526040908101905180910390a1505050505050565b600160a060020a039081166000908152600160205260409020541690565b60075481565b60085481565b600160a060020a031660009081526001602052604090205460a060020a900460ff1690565b60055481565b60065481565b60008282018381101561108c57fe5b9392505050565b60008282111561109f57fe5b50900390565b8154818355818115116110c9576000838152602090206110c99181019083016110e0565b505050565b60206040519081016040526000815290565b6108af91905b808211156110fa57600081556001016110e6565b50905600a165627a7a72305820555de7c5131842a4fccb258fccd95ae1539019bb744b4253893b37fed1b3d8e90029"

// DeployValidator deploys a new Ethereum contract, binding an instance of Validator to it.
func DeployValidator(auth *bind.TransactOpts, backend bind.ContractBackend, _candidates []common.Address, _caps []*big.Int, _firstOwner common.Address, _minCandidateCap *big.Int, _minVoterCap *big.Int, _maxValidatorNumber *big.Int, _candidateWithdrawDelay *big.Int, _voterWithdrawDelay *big.Int) (common.Address, *types.Transaction, *Validator, error) {
	parsed, err := abi.JSON(strings.NewReader(ValidatorABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}

	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ValidatorBin), backend, _candidates, _caps, _firstOwner, _minCandidateCap, _minVoterCap, _maxValidatorNumber, _candidateWithdrawDelay, _voterWithdrawDelay)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Validator{ValidatorCaller: ValidatorCaller{contract: contract}, ValidatorTransactor: ValidatorTransactor{contract: contract}, ValidatorFilterer: ValidatorFilterer{contract: contract}}, nil
}

// Validator is an auto generated Go binding around an Ethereum contract.
type Validator struct {
	ValidatorCaller     // Read-only binding to the contract
	ValidatorTransactor // Write-only binding to the contract
	ValidatorFilterer   // Log filterer for contract events
}

// ValidatorCaller is an auto generated read-only Go binding around an Ethereum contract.
type ValidatorCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ValidatorTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ValidatorFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ValidatorSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ValidatorSession struct {
	Contract     *Validator        // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ValidatorCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ValidatorCallerSession struct {
	Contract *ValidatorCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts    // Call options to use throughout this session
}

// ValidatorTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ValidatorTransactorSession struct {
	Contract     *ValidatorTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts    // Transaction auth options to use throughout this session
}

// ValidatorRaw is an auto generated low-level Go binding around an Ethereum contract.
type ValidatorRaw struct {
	Contract *Validator // Generic contract binding to access the raw methods on
}

// ValidatorCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ValidatorCallerRaw struct {
	Contract *ValidatorCaller // Generic read-only contract binding to access the raw methods on
}

// ValidatorTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ValidatorTransactorRaw struct {
	Contract *ValidatorTransactor // Generic write-only contract binding to access the raw methods on
}

// NewValidator creates a new instance of Validator, bound to a specific deployed contract.
func NewValidator(address common.Address, backend bind.ContractBackend) (*Validator, error) {
	contract, err := bindValidator(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Validator{ValidatorCaller: ValidatorCaller{contract: contract}, ValidatorTransactor: ValidatorTransactor{contract: contract}, ValidatorFilterer: ValidatorFilterer{contract: contract}}, nil
}

// NewValidatorCaller creates a new read-only instance of Validator, bound to a specific deployed contract.
func NewValidatorCaller(address common.Address, caller bind.ContractCaller) (*ValidatorCaller, error) {
	contract, err := bindValidator(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ValidatorCaller{contract: contract}, nil
}

// NewValidatorTransactor creates a new write-only instance of Validator, bound to a specific deployed contract.
func NewValidatorTransactor(address common.Address, transactor bind.ContractTransactor) (*ValidatorTransactor, error) {
	contract, err := bindValidator(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ValidatorTransactor{contract: contract}, nil
}

// NewValidatorFilterer creates a new log filterer instance of Validator, bound to a specific deployed contract.
func NewValidatorFilterer(address common.Address, filterer bind.ContractFilterer) (*ValidatorFilterer, error) {
	contract, err := bindValidator(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ValidatorFilterer{contract: contract}, nil
}

// bindValidator binds a generic wrapper to an already deployed contract.
func bindValidator(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ValidatorABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Validator *ValidatorRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Validator.Contract.ValidatorCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Validator *ValidatorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Validator.Contract.ValidatorTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Validator *ValidatorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Validator.Contract.ValidatorTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Validator *ValidatorCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Validator.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Validator *ValidatorTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Validator.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Validator *ValidatorTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Validator.Contract.contract.Transact(opts, method, params...)
}

// CandidateCount is a free data retrieval call binding the contract method 0xa9a981a3.
//
// Solidity: function candidateCount() view returns(uint256)
func (_Validator *ValidatorCaller) CandidateCount(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "candidateCount")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CandidateCount is a free data retrieval call binding the contract method 0xa9a981a3.
//
// Solidity: function candidateCount() view returns(uint256)
func (_Validator *ValidatorSession) CandidateCount() (*big.Int, error) {
	return _Validator.Contract.CandidateCount(&_Validator.CallOpts)
}

// CandidateCount is a free data retrieval call binding the contract method 0xa9a981a3.
//
// Solidity: function candidateCount() view returns(uint256)
func (_Validator *ValidatorCallerSession) CandidateCount() (*big.Int, error) {
	return _Validator.Contract.CandidateCount(&_Validator.CallOpts)
}

// CandidateWithdrawDelay is a free data retrieval call binding the contract method 0xd161c767.
//
// Solidity: function candidateWithdrawDelay() view returns(uint256)
func (_Validator *ValidatorCaller) CandidateWithdrawDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "candidateWithdrawDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// CandidateWithdrawDelay is a free data retrieval call binding the contract method 0xd161c767.
//
// Solidity: function candidateWithdrawDelay() view returns(uint256)
func (_Validator *ValidatorSession) CandidateWithdrawDelay() (*big.Int, error) {
	return _Validator.Contract.CandidateWithdrawDelay(&_Validator.CallOpts)
}

// CandidateWithdrawDelay is a free data retrieval call binding the contract method 0xd161c767.
//
// Solidity: function candidateWithdrawDelay() view returns(uint256)
func (_Validator *ValidatorCallerSession) CandidateWithdrawDelay() (*big.Int, error) {
	return _Validator.Contract.CandidateWithdrawDelay(&_Validator.CallOpts)
}

// Candidates is a free data retrieval call binding the contract method 0x3477ee2e.
//
// Solidity: function candidates(uint256 ) view returns(address)
func (_Validator *ValidatorCaller) Candidates(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "candidates", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Candidates is a free data retrieval call binding the contract method 0x3477ee2e.
//
// Solidity: function candidates(uint256 ) view returns(address)
func (_Validator *ValidatorSession) Candidates(arg0 *big.Int) (common.Address, error) {
	return _Validator.Contract.Candidates(&_Validator.CallOpts, arg0)
}

// Candidates is a free data retrieval call binding the contract method 0x3477ee2e.
//
// Solidity: function candidates(uint256 ) view returns(address)
func (_Validator *ValidatorCallerSession) Candidates(arg0 *big.Int) (common.Address, error) {
	return _Validator.Contract.Candidates(&_Validator.CallOpts, arg0)
}

// GetCandidateCap is a free data retrieval call binding the contract method 0x58e7525f.
//
// Solidity: function getCandidateCap(address _candidate) view returns(uint256)
func (_Validator *ValidatorCaller) GetCandidateCap(opts *bind.CallOpts, _candidate common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "getCandidateCap", _candidate)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCandidateCap is a free data retrieval call binding the contract method 0x58e7525f.
//
// Solidity: function getCandidateCap(address _candidate) view returns(uint256)
func (_Validator *ValidatorSession) GetCandidateCap(_candidate common.Address) (*big.Int, error) {
	return _Validator.Contract.GetCandidateCap(&_Validator.CallOpts, _candidate)
}

// GetCandidateCap is a free data retrieval call binding the contract method 0x58e7525f.
//
// Solidity: function getCandidateCap(address _candidate) view returns(uint256)
func (_Validator *ValidatorCallerSession) GetCandidateCap(_candidate common.Address) (*big.Int, error) {
	return _Validator.Contract.GetCandidateCap(&_Validator.CallOpts, _candidate)
}

// GetCandidateOwner is a free data retrieval call binding the contract method 0xb642facd.
//
// Solidity: function getCandidateOwner(address _candidate) view returns(address)
func (_Validator *ValidatorCaller) GetCandidateOwner(opts *bind.CallOpts, _candidate common.Address) (common.Address, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "getCandidateOwner", _candidate)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetCandidateOwner is a free data retrieval call binding the contract method 0xb642facd.
//
// Solidity: function getCandidateOwner(address _candidate) view returns(address)
func (_Validator *ValidatorSession) GetCandidateOwner(_candidate common.Address) (common.Address, error) {
	return _Validator.Contract.GetCandidateOwner(&_Validator.CallOpts, _candidate)
}

// GetCandidateOwner is a free data retrieval call binding the contract method 0xb642facd.
//
// Solidity: function getCandidateOwner(address _candidate) view returns(address)
func (_Validator *ValidatorCallerSession) GetCandidateOwner(_candidate common.Address) (common.Address, error) {
	return _Validator.Contract.GetCandidateOwner(&_Validator.CallOpts, _candidate)
}

// GetCandidates is a free data retrieval call binding the contract method 0x06a49fce.
//
// Solidity: function getCandidates() view returns(address[])
func (_Validator *ValidatorCaller) GetCandidates(opts *bind.CallOpts) ([]common.Address, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "getCandidates")

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetCandidates is a free data retrieval call binding the contract method 0x06a49fce.
//
// Solidity: function getCandidates() view returns(address[])
func (_Validator *ValidatorSession) GetCandidates() ([]common.Address, error) {
	return _Validator.Contract.GetCandidates(&_Validator.CallOpts)
}

// GetCandidates is a free data retrieval call binding the contract method 0x06a49fce.
//
// Solidity: function getCandidates() view returns(address[])
func (_Validator *ValidatorCallerSession) GetCandidates() ([]common.Address, error) {
	return _Validator.Contract.GetCandidates(&_Validator.CallOpts)
}

// GetVoterCap is a free data retrieval call binding the contract method 0x302b6872.
//
// Solidity: function getVoterCap(address _candidate, address _voter) view returns(uint256)
func (_Validator *ValidatorCaller) GetVoterCap(opts *bind.CallOpts, _candidate common.Address, _voter common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "getVoterCap", _candidate, _voter)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetVoterCap is a free data retrieval call binding the contract method 0x302b6872.
//
// Solidity: function getVoterCap(address _candidate, address _voter) view returns(uint256)
func (_Validator *ValidatorSession) GetVoterCap(_candidate common.Address, _voter common.Address) (*big.Int, error) {
	return _Validator.Contract.GetVoterCap(&_Validator.CallOpts, _candidate, _voter)
}

// GetVoterCap is a free data retrieval call binding the contract method 0x302b6872.
//
// Solidity: function getVoterCap(address _candidate, address _voter) view returns(uint256)
func (_Validator *ValidatorCallerSession) GetVoterCap(_candidate common.Address, _voter common.Address) (*big.Int, error) {
	return _Validator.Contract.GetVoterCap(&_Validator.CallOpts, _candidate, _voter)
}

// GetVoters is a free data retrieval call binding the contract method 0x2d15cc04.
//
// Solidity: function getVoters(address _candidate) view returns(address[])
func (_Validator *ValidatorCaller) GetVoters(opts *bind.CallOpts, _candidate common.Address) ([]common.Address, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "getVoters", _candidate)

	if err != nil {
		return *new([]common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new([]common.Address)).(*[]common.Address)

	return out0, err

}

// GetVoters is a free data retrieval call binding the contract method 0x2d15cc04.
//
// Solidity: function getVoters(address _candidate) view returns(address[])
func (_Validator *ValidatorSession) GetVoters(_candidate common.Address) ([]common.Address, error) {
	return _Validator.Contract.GetVoters(&_Validator.CallOpts, _candidate)
}

// GetVoters is a free data retrieval call binding the contract method 0x2d15cc04.
//
// Solidity: function getVoters(address _candidate) view returns(address[])
func (_Validator *ValidatorCallerSession) GetVoters(_candidate common.Address) ([]common.Address, error) {
	return _Validator.Contract.GetVoters(&_Validator.CallOpts, _candidate)
}

// GetWithdrawBlockNumbers is a free data retrieval call binding the contract method 0x2f9c4bba.
//
// Solidity: function getWithdrawBlockNumbers() view returns(uint256[])
func (_Validator *ValidatorCaller) GetWithdrawBlockNumbers(opts *bind.CallOpts) ([]*big.Int, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "getWithdrawBlockNumbers")

	if err != nil {
		return *new([]*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new([]*big.Int)).(*[]*big.Int)

	return out0, err

}

// GetWithdrawBlockNumbers is a free data retrieval call binding the contract method 0x2f9c4bba.
//
// Solidity: function getWithdrawBlockNumbers() view returns(uint256[])
func (_Validator *ValidatorSession) GetWithdrawBlockNumbers() ([]*big.Int, error) {
	return _Validator.Contract.GetWithdrawBlockNumbers(&_Validator.CallOpts)
}

// GetWithdrawBlockNumbers is a free data retrieval call binding the contract method 0x2f9c4bba.
//
// Solidity: function getWithdrawBlockNumbers() view returns(uint256[])
func (_Validator *ValidatorCallerSession) GetWithdrawBlockNumbers() ([]*big.Int, error) {
	return _Validator.Contract.GetWithdrawBlockNumbers(&_Validator.CallOpts)
}

// GetWithdrawCap is a free data retrieval call binding the contract method 0x15febd68.
//
// Solidity: function getWithdrawCap(uint256 _blockNumber) view returns(uint256)
func (_Validator *ValidatorCaller) GetWithdrawCap(opts *bind.CallOpts, _blockNumber *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "getWithdrawCap", _blockNumber)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetWithdrawCap is a free data retrieval call binding the contract method 0x15febd68.
//
// Solidity: function getWithdrawCap(uint256 _blockNumber) view returns(uint256)
func (_Validator *ValidatorSession) GetWithdrawCap(_blockNumber *big.Int) (*big.Int, error) {
	return _Validator.Contract.GetWithdrawCap(&_Validator.CallOpts, _blockNumber)
}

// GetWithdrawCap is a free data retrieval call binding the contract method 0x15febd68.
//
// Solidity: function getWithdrawCap(uint256 _blockNumber) view returns(uint256)
func (_Validator *ValidatorCallerSession) GetWithdrawCap(_blockNumber *big.Int) (*big.Int, error) {
	return _Validator.Contract.GetWithdrawCap(&_Validator.CallOpts, _blockNumber)
}

// IsCandidate is a free data retrieval call binding the contract method 0xd51b9e93.
//
// Solidity: function isCandidate(address _candidate) view returns(bool)
func (_Validator *ValidatorCaller) IsCandidate(opts *bind.CallOpts, _candidate common.Address) (bool, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "isCandidate", _candidate)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsCandidate is a free data retrieval call binding the contract method 0xd51b9e93.
//
// Solidity: function isCandidate(address _candidate) view returns(bool)
func (_Validator *ValidatorSession) IsCandidate(_candidate common.Address) (bool, error) {
	return _Validator.Contract.IsCandidate(&_Validator.CallOpts, _candidate)
}

// IsCandidate is a free data retrieval call binding the contract method 0xd51b9e93.
//
// Solidity: function isCandidate(address _candidate) view returns(bool)
func (_Validator *ValidatorCallerSession) IsCandidate(_candidate common.Address) (bool, error) {
	return _Validator.Contract.IsCandidate(&_Validator.CallOpts, _candidate)
}

// MaxValidatorNumber is a free data retrieval call binding the contract method 0xd09f1ab4.
//
// Solidity: function maxValidatorNumber() view returns(uint256)
func (_Validator *ValidatorCaller) MaxValidatorNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "maxValidatorNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxValidatorNumber is a free data retrieval call binding the contract method 0xd09f1ab4.
//
// Solidity: function maxValidatorNumber() view returns(uint256)
func (_Validator *ValidatorSession) MaxValidatorNumber() (*big.Int, error) {
	return _Validator.Contract.MaxValidatorNumber(&_Validator.CallOpts)
}

// MaxValidatorNumber is a free data retrieval call binding the contract method 0xd09f1ab4.
//
// Solidity: function maxValidatorNumber() view returns(uint256)
func (_Validator *ValidatorCallerSession) MaxValidatorNumber() (*big.Int, error) {
	return _Validator.Contract.MaxValidatorNumber(&_Validator.CallOpts)
}

// MinCandidateCap is a free data retrieval call binding the contract method 0xd55b7dff.
//
// Solidity: function minCandidateCap() view returns(uint256)
func (_Validator *ValidatorCaller) MinCandidateCap(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "minCandidateCap")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinCandidateCap is a free data retrieval call binding the contract method 0xd55b7dff.
//
// Solidity: function minCandidateCap() view returns(uint256)
func (_Validator *ValidatorSession) MinCandidateCap() (*big.Int, error) {
	return _Validator.Contract.MinCandidateCap(&_Validator.CallOpts)
}

// MinCandidateCap is a free data retrieval call binding the contract method 0xd55b7dff.
//
// Solidity: function minCandidateCap() view returns(uint256)
func (_Validator *ValidatorCallerSession) MinCandidateCap() (*big.Int, error) {
	return _Validator.Contract.MinCandidateCap(&_Validator.CallOpts)
}

// MinVoterCap is a free data retrieval call binding the contract method 0xf8ac9dd5.
//
// Solidity: function minVoterCap() view returns(uint256)
func (_Validator *ValidatorCaller) MinVoterCap(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "minVoterCap")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MinVoterCap is a free data retrieval call binding the contract method 0xf8ac9dd5.
//
// Solidity: function minVoterCap() view returns(uint256)
func (_Validator *ValidatorSession) MinVoterCap() (*big.Int, error) {
	return _Validator.Contract.MinVoterCap(&_Validator.CallOpts)
}

// MinVoterCap is a free data retrieval call binding the contract method 0xf8ac9dd5.
//
// Solidity: function minVoterCap() view returns(uint256)
func (_Validator *ValidatorCallerSession) MinVoterCap() (*big.Int, error) {
	return _Validator.Contract.MinVoterCap(&_Validator.CallOpts)
}

// VoterWithdrawDelay is a free data retrieval call binding the contract method 0xa9ff959e.
//
// Solidity: function voterWithdrawDelay() view returns(uint256)
func (_Validator *ValidatorCaller) VoterWithdrawDelay(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Validator.contract.Call(opts, &out, "voterWithdrawDelay")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VoterWithdrawDelay is a free data retrieval call binding the contract method 0xa9ff959e.
//
// Solidity: function voterWithdrawDelay() view returns(uint256)
func (_Validator *ValidatorSession) VoterWithdrawDelay() (*big.Int, error) {
	return _Validator.Contract.VoterWithdrawDelay(&_Validator.CallOpts)
}

// VoterWithdrawDelay is a free data retrieval call binding the contract method 0xa9ff959e.
//
// Solidity: function voterWithdrawDelay() view returns(uint256)
func (_Validator *ValidatorCallerSession) VoterWithdrawDelay() (*big.Int, error) {
	return _Validator.Contract.VoterWithdrawDelay(&_Validator.CallOpts)
}

// Propose is a paid mutator transaction binding the contract method 0x01267951.
//
// Solidity: function propose(address _candidate) payable returns()
func (_Validator *ValidatorTransactor) Propose(opts *bind.TransactOpts, _candidate common.Address) (*types.Transaction, error) {
	return _Validator.contract.Transact(opts, "propose", _candidate)
}

// Propose is a paid mutator transaction binding the contract method 0x01267951.
//
// Solidity: function propose(address _candidate) payable returns()
func (_Validator *ValidatorSession) Propose(_candidate common.Address) (*types.Transaction, error) {
	return _Validator.Contract.Propose(&_Validator.TransactOpts, _candidate)
}

// Propose is a paid mutator transaction binding the contract method 0x01267951.
//
// Solidity: function propose(address _candidate) payable returns()
func (_Validator *ValidatorTransactorSession) Propose(_candidate common.Address) (*types.Transaction, error) {
	return _Validator.Contract.Propose(&_Validator.TransactOpts, _candidate)
}

// Resign is a paid mutator transaction binding the contract method 0xae6e43f5.
//
// Solidity: function resign(address _candidate) returns()
func (_Validator *ValidatorTransactor) Resign(opts *bind.TransactOpts, _candidate common.Address) (*types.Transaction, error) {
	return _Validator.contract.Transact(opts, "resign", _candidate)
}

// Resign is a paid mutator transaction binding the contract method 0xae6e43f5.
//
// Solidity: function resign(address _candidate) returns()
func (_Validator *ValidatorSession) Resign(_candidate common.Address) (*types.Transaction, error) {
	return _Validator.Contract.Resign(&_Validator.TransactOpts, _candidate)
}

// Resign is a paid mutator transaction binding the contract method 0xae6e43f5.
//
// Solidity: function resign(address _candidate) returns()
func (_Validator *ValidatorTransactorSession) Resign(_candidate common.Address) (*types.Transaction, error) {
	return _Validator.Contract.Resign(&_Validator.TransactOpts, _candidate)
}

// Unvote is a paid mutator transaction binding the contract method 0x02aa9be2.
//
// Solidity: function unvote(address _candidate, uint256 _cap) returns()
func (_Validator *ValidatorTransactor) Unvote(opts *bind.TransactOpts, _candidate common.Address, _cap *big.Int) (*types.Transaction, error) {
	return _Validator.contract.Transact(opts, "unvote", _candidate, _cap)
}

// Unvote is a paid mutator transaction binding the contract method 0x02aa9be2.
//
// Solidity: function unvote(address _candidate, uint256 _cap) returns()
func (_Validator *ValidatorSession) Unvote(_candidate common.Address, _cap *big.Int) (*types.Transaction, error) {
	return _Validator.Contract.Unvote(&_Validator.TransactOpts, _candidate, _cap)
}

// Unvote is a paid mutator transaction binding the contract method 0x02aa9be2.
//
// Solidity: function unvote(address _candidate, uint256 _cap) returns()
func (_Validator *ValidatorTransactorSession) Unvote(_candidate common.Address, _cap *big.Int) (*types.Transaction, error) {
	return _Validator.Contract.Unvote(&_Validator.TransactOpts, _candidate, _cap)
}

// Vote is a paid mutator transaction binding the contract method 0x6dd7d8ea.
//
// Solidity: function vote(address _candidate) payable returns()
func (_Validator *ValidatorTransactor) Vote(opts *bind.TransactOpts, _candidate common.Address) (*types.Transaction, error) {
	return _Validator.contract.Transact(opts, "vote", _candidate)
}

// Vote is a paid mutator transaction binding the contract method 0x6dd7d8ea.
//
// Solidity: function vote(address _candidate) payable returns()
func (_Validator *ValidatorSession) Vote(_candidate common.Address) (*types.Transaction, error) {
	return _Validator.Contract.Vote(&_Validator.TransactOpts, _candidate)
}

// Vote is a paid mutator transaction binding the contract method 0x6dd7d8ea.
//
// Solidity: function vote(address _candidate) payable returns()
func (_Validator *ValidatorTransactorSession) Vote(_candidate common.Address) (*types.Transaction, error) {
	return _Validator.Contract.Vote(&_Validator.TransactOpts, _candidate)
}

// Withdraw is a paid mutator transaction binding the contract method 0x441a3e70.
//
// Solidity: function withdraw(uint256 _blockNumber, uint256 _index) returns()
func (_Validator *ValidatorTransactor) Withdraw(opts *bind.TransactOpts, _blockNumber *big.Int, _index *big.Int) (*types.Transaction, error) {
	return _Validator.contract.Transact(opts, "withdraw", _blockNumber, _index)
}

// Withdraw is a paid mutator transaction binding the contract method 0x441a3e70.
//
// Solidity: function withdraw(uint256 _blockNumber, uint256 _index) returns()
func (_Validator *ValidatorSession) Withdraw(_blockNumber *big.Int, _index *big.Int) (*types.Transaction, error) {
	return _Validator.Contract.Withdraw(&_Validator.TransactOpts, _blockNumber, _index)
}

// Withdraw is a paid mutator transaction binding the contract method 0x441a3e70.
//
// Solidity: function withdraw(uint256 _blockNumber, uint256 _index) returns()
func (_Validator *ValidatorTransactorSession) Withdraw(_blockNumber *big.Int, _index *big.Int) (*types.Transaction, error) {
	return _Validator.Contract.Withdraw(&_Validator.TransactOpts, _blockNumber, _index)
}

// ValidatorProposeIterator is returned from FilterPropose and is used to iterate over the raw logs and unpacked data for Propose events raised by the Validator contract.
type ValidatorProposeIterator struct {
	Event *ValidatorPropose // Event containing the contract specifics and raw log

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
func (it *ValidatorProposeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidatorPropose)
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
		it.Event = new(ValidatorPropose)
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
func (it *ValidatorProposeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidatorProposeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidatorPropose represents a Propose event raised by the Validator contract.
type ValidatorPropose struct {
	Owner     common.Address
	Candidate common.Address
	Cap       *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterPropose is a free log retrieval operation binding the contract event 0x7635f1d87b47fba9f2b09e56eb4be75cca030e0cb179c1602ac9261d39a8f5c1.
//
// Solidity: event Propose(address _owner, address _candidate, uint256 _cap)
func (_Validator *ValidatorFilterer) FilterPropose(opts *bind.FilterOpts) (*ValidatorProposeIterator, error) {

	logs, sub, err := _Validator.contract.FilterLogs(opts, "Propose")
	if err != nil {
		return nil, err
	}
	return &ValidatorProposeIterator{contract: _Validator.contract, event: "Propose", logs: logs, sub: sub}, nil
}

// WatchPropose is a free log subscription operation binding the contract event 0x7635f1d87b47fba9f2b09e56eb4be75cca030e0cb179c1602ac9261d39a8f5c1.
//
// Solidity: event Propose(address _owner, address _candidate, uint256 _cap)
func (_Validator *ValidatorFilterer) WatchPropose(opts *bind.WatchOpts, sink chan<- *ValidatorPropose) (event.Subscription, error) {

	logs, sub, err := _Validator.contract.WatchLogs(opts, "Propose")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidatorPropose)
				if err := _Validator.contract.UnpackLog(event, "Propose", log); err != nil {
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

// ParsePropose is a log parse operation binding the contract event 0x7635f1d87b47fba9f2b09e56eb4be75cca030e0cb179c1602ac9261d39a8f5c1.
//
// Solidity: event Propose(address _owner, address _candidate, uint256 _cap)
func (_Validator *ValidatorFilterer) ParsePropose(log types.Log) (*ValidatorPropose, error) {
	event := new(ValidatorPropose)
	if err := _Validator.contract.UnpackLog(event, "Propose", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ValidatorResignIterator is returned from FilterResign and is used to iterate over the raw logs and unpacked data for Resign events raised by the Validator contract.
type ValidatorResignIterator struct {
	Event *ValidatorResign // Event containing the contract specifics and raw log

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
func (it *ValidatorResignIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidatorResign)
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
		it.Event = new(ValidatorResign)
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
func (it *ValidatorResignIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidatorResignIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidatorResign represents a Resign event raised by the Validator contract.
type ValidatorResign struct {
	Owner     common.Address
	Candidate common.Address
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterResign is a free log retrieval operation binding the contract event 0x4edf3e325d0063213a39f9085522994a1c44bea5f39e7d63ef61260a1e58c6d3.
//
// Solidity: event Resign(address _owner, address _candidate)
func (_Validator *ValidatorFilterer) FilterResign(opts *bind.FilterOpts) (*ValidatorResignIterator, error) {

	logs, sub, err := _Validator.contract.FilterLogs(opts, "Resign")
	if err != nil {
		return nil, err
	}
	return &ValidatorResignIterator{contract: _Validator.contract, event: "Resign", logs: logs, sub: sub}, nil
}

// WatchResign is a free log subscription operation binding the contract event 0x4edf3e325d0063213a39f9085522994a1c44bea5f39e7d63ef61260a1e58c6d3.
//
// Solidity: event Resign(address _owner, address _candidate)
func (_Validator *ValidatorFilterer) WatchResign(opts *bind.WatchOpts, sink chan<- *ValidatorResign) (event.Subscription, error) {

	logs, sub, err := _Validator.contract.WatchLogs(opts, "Resign")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidatorResign)
				if err := _Validator.contract.UnpackLog(event, "Resign", log); err != nil {
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

// ParseResign is a log parse operation binding the contract event 0x4edf3e325d0063213a39f9085522994a1c44bea5f39e7d63ef61260a1e58c6d3.
//
// Solidity: event Resign(address _owner, address _candidate)
func (_Validator *ValidatorFilterer) ParseResign(log types.Log) (*ValidatorResign, error) {
	event := new(ValidatorResign)
	if err := _Validator.contract.UnpackLog(event, "Resign", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ValidatorUnvoteIterator is returned from FilterUnvote and is used to iterate over the raw logs and unpacked data for Unvote events raised by the Validator contract.
type ValidatorUnvoteIterator struct {
	Event *ValidatorUnvote // Event containing the contract specifics and raw log

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
func (it *ValidatorUnvoteIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidatorUnvote)
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
		it.Event = new(ValidatorUnvote)
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
func (it *ValidatorUnvoteIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidatorUnvoteIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidatorUnvote represents a Unvote event raised by the Validator contract.
type ValidatorUnvote struct {
	Voter     common.Address
	Candidate common.Address
	Cap       *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterUnvote is a free log retrieval operation binding the contract event 0xaa0e554f781c3c3b2be110a0557f260f11af9a8aa2c64bc1e7a31dbb21e32fa2.
//
// Solidity: event Unvote(address _voter, address _candidate, uint256 _cap)
func (_Validator *ValidatorFilterer) FilterUnvote(opts *bind.FilterOpts) (*ValidatorUnvoteIterator, error) {

	logs, sub, err := _Validator.contract.FilterLogs(opts, "Unvote")
	if err != nil {
		return nil, err
	}
	return &ValidatorUnvoteIterator{contract: _Validator.contract, event: "Unvote", logs: logs, sub: sub}, nil
}

// WatchUnvote is a free log subscription operation binding the contract event 0xaa0e554f781c3c3b2be110a0557f260f11af9a8aa2c64bc1e7a31dbb21e32fa2.
//
// Solidity: event Unvote(address _voter, address _candidate, uint256 _cap)
func (_Validator *ValidatorFilterer) WatchUnvote(opts *bind.WatchOpts, sink chan<- *ValidatorUnvote) (event.Subscription, error) {

	logs, sub, err := _Validator.contract.WatchLogs(opts, "Unvote")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidatorUnvote)
				if err := _Validator.contract.UnpackLog(event, "Unvote", log); err != nil {
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

// ParseUnvote is a log parse operation binding the contract event 0xaa0e554f781c3c3b2be110a0557f260f11af9a8aa2c64bc1e7a31dbb21e32fa2.
//
// Solidity: event Unvote(address _voter, address _candidate, uint256 _cap)
func (_Validator *ValidatorFilterer) ParseUnvote(log types.Log) (*ValidatorUnvote, error) {
	event := new(ValidatorUnvote)
	if err := _Validator.contract.UnpackLog(event, "Unvote", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ValidatorVoteIterator is returned from FilterVote and is used to iterate over the raw logs and unpacked data for Vote events raised by the Validator contract.
type ValidatorVoteIterator struct {
	Event *ValidatorVote // Event containing the contract specifics and raw log

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
func (it *ValidatorVoteIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidatorVote)
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
		it.Event = new(ValidatorVote)
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
func (it *ValidatorVoteIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidatorVoteIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidatorVote represents a Vote event raised by the Validator contract.
type ValidatorVote struct {
	Voter     common.Address
	Candidate common.Address
	Cap       *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterVote is a free log retrieval operation binding the contract event 0x66a9138482c99e9baf08860110ef332cc0c23b4a199a53593d8db0fc8f96fbfc.
//
// Solidity: event Vote(address _voter, address _candidate, uint256 _cap)
func (_Validator *ValidatorFilterer) FilterVote(opts *bind.FilterOpts) (*ValidatorVoteIterator, error) {

	logs, sub, err := _Validator.contract.FilterLogs(opts, "Vote")
	if err != nil {
		return nil, err
	}
	return &ValidatorVoteIterator{contract: _Validator.contract, event: "Vote", logs: logs, sub: sub}, nil
}

// WatchVote is a free log subscription operation binding the contract event 0x66a9138482c99e9baf08860110ef332cc0c23b4a199a53593d8db0fc8f96fbfc.
//
// Solidity: event Vote(address _voter, address _candidate, uint256 _cap)
func (_Validator *ValidatorFilterer) WatchVote(opts *bind.WatchOpts, sink chan<- *ValidatorVote) (event.Subscription, error) {

	logs, sub, err := _Validator.contract.WatchLogs(opts, "Vote")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidatorVote)
				if err := _Validator.contract.UnpackLog(event, "Vote", log); err != nil {
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

// ParseVote is a log parse operation binding the contract event 0x66a9138482c99e9baf08860110ef332cc0c23b4a199a53593d8db0fc8f96fbfc.
//
// Solidity: event Vote(address _voter, address _candidate, uint256 _cap)
func (_Validator *ValidatorFilterer) ParseVote(log types.Log) (*ValidatorVote, error) {
	event := new(ValidatorVote)
	if err := _Validator.contract.UnpackLog(event, "Vote", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ValidatorWithdrawIterator is returned from FilterWithdraw and is used to iterate over the raw logs and unpacked data for Withdraw events raised by the Validator contract.
type ValidatorWithdrawIterator struct {
	Event *ValidatorWithdraw // Event containing the contract specifics and raw log

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
func (it *ValidatorWithdrawIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ValidatorWithdraw)
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
		it.Event = new(ValidatorWithdraw)
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
func (it *ValidatorWithdrawIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ValidatorWithdrawIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ValidatorWithdraw represents a Withdraw event raised by the Validator contract.
type ValidatorWithdraw struct {
	Owner       common.Address
	BlockNumber *big.Int
	Cap         *big.Int
	Raw         types.Log // Blockchain specific contextual infos
}

// FilterWithdraw is a free log retrieval operation binding the contract event 0xf279e6a1f5e320cca91135676d9cb6e44ca8a08c0b88342bcdb1144f6511b568.
//
// Solidity: event Withdraw(address _owner, uint256 _blockNumber, uint256 _cap)
func (_Validator *ValidatorFilterer) FilterWithdraw(opts *bind.FilterOpts) (*ValidatorWithdrawIterator, error) {

	logs, sub, err := _Validator.contract.FilterLogs(opts, "Withdraw")
	if err != nil {
		return nil, err
	}
	return &ValidatorWithdrawIterator{contract: _Validator.contract, event: "Withdraw", logs: logs, sub: sub}, nil
}

// WatchWithdraw is a free log subscription operation binding the contract event 0xf279e6a1f5e320cca91135676d9cb6e44ca8a08c0b88342bcdb1144f6511b568.
//
// Solidity: event Withdraw(address _owner, uint256 _blockNumber, uint256 _cap)
func (_Validator *ValidatorFilterer) WatchWithdraw(opts *bind.WatchOpts, sink chan<- *ValidatorWithdraw) (event.Subscription, error) {

	logs, sub, err := _Validator.contract.WatchLogs(opts, "Withdraw")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ValidatorWithdraw)
				if err := _Validator.contract.UnpackLog(event, "Withdraw", log); err != nil {
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

// ParseWithdraw is a log parse operation binding the contract event 0xf279e6a1f5e320cca91135676d9cb6e44ca8a08c0b88342bcdb1144f6511b568.
//
// Solidity: event Withdraw(address _owner, uint256 _blockNumber, uint256 _cap)
func (_Validator *ValidatorFilterer) ParseWithdraw(log types.Log) (*ValidatorWithdraw, error) {
	event := new(ValidatorWithdraw)
	if err := _Validator.contract.UnpackLog(event, "Withdraw", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
