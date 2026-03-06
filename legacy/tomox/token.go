package tomox

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/contracts/tomox/contract"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// GetTokenAbi return token abi
func GetTokenAbi() (*abi.ABI, error) {
	contractABI, err := abi.JSON(strings.NewReader(contract.TRC21ABI))
	if err != nil {
		return nil, err
	}
	return &contractABI, nil
}

// RunContract executes a read-only smart contract call using the EVM directly.
// This replaces the legacy victionchain approach that used SimulatedBackend.CallContractWithState.
func RunContract(chain consensus.ChainContext, statedb *state.StateDB, contractAddr common.Address, abi *abi.ABI, method string, args ...interface{}) (interface{}, error) {
	input, err := abi.Pack(method, args...)
	if err != nil {
		return nil, err
	}

	fakeCaller := common.HexToAddress("0x0000000000000000000000000000000000000001")

	header := chain.CurrentHeader()
	chainConfig := chain.Config()
	if chainConfig == nil {
		chainConfig = params.VictionChainConfig
	}

	// Create a minimal EVM context for a read-only call
	blockCtx := vm.BlockContext{
		CanTransfer: func(db vm.StateDB, addr common.Address, amount *big.Int) bool { return true },
		Transfer:    func(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {},
		GetHash:     func(n uint64) common.Hash { return common.Hash{} },
		Coinbase:    common.Address{},
		BlockNumber: new(big.Int).Set(header.Number),
		Time:        new(big.Int).SetUint64(header.Time),
		GasLimit:    1000000,
		Difficulty:  new(big.Int).Set(header.Difficulty),
	}
	txCtx := vm.TxContext{
		Origin:   fakeCaller,
		GasPrice: big.NewInt(0),
	}

	evm := vm.NewEVM(blockCtx, txCtx, statedb, chainConfig, vm.Config{})
	result, _, err := evm.StaticCall(vm.AccountRef(fakeCaller), contractAddr, input, 1000000)
	if err != nil {
		return nil, err
	}

	var unpackResult interface{}
	err = abi.UnpackIntoInterface(&unpackResult, method, result)
	if err != nil {
		return nil, err
	}
	return unpackResult, nil
}

func (tomox *TomoX) GetTokenDecimal(chain consensus.ChainContext, statedb *state.StateDB, tokenAddr common.Address) (*big.Int, error) {
	if tokenDecimal, ok := tomox.tokenDecimalCache.Get(tokenAddr); ok {
		return tokenDecimal.(*big.Int), nil
	}
	if tokenAddr.String() == params.TomoNativeAddress {
		tomox.tokenDecimalCache.Add(tokenAddr, params.BasePrice)
		return params.BasePrice, nil
	}
	var decimals uint8
	defer func() {
		log.Debug("GetTokenDecimal from ", "relayerSMC", params.RelayerRegistrationSMC, "tokenAddr", tokenAddr.Hex(), "decimals", decimals)
	}()
	contractABI, err := GetTokenAbi()
	if err != nil {
		return nil, err
	}
	stateCopy := statedb.Copy()
	result, err := RunContract(chain, stateCopy, tokenAddr, contractABI, "decimals")
	if err != nil {
		return nil, err
	}
	decimals = result.(uint8)

	tokenDecimal := new(big.Int).SetUint64(0).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	tomox.tokenDecimalCache.Add(tokenAddr, tokenDecimal)
	return tokenDecimal, nil
}

// FIXME: using in unit tests only
func (tomox *TomoX) SetTokenDecimal(token common.Address, decimal *big.Int) {
	tomox.tokenDecimalCache.Add(token, decimal)
}
