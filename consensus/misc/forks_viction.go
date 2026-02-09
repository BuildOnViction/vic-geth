package misc

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vrc25"
	"github.com/ethereum/go-ethereum/params"
)

// ApplySaigonHardFork mint additional token to EcoSystem Multisig preiodly for 4 years
func ApplySaigonHardFork(statedb *state.StateDB, config *params.VictionConfig, saigonBlock *big.Int, headBlock *big.Int) {
	if config.SaigonFundInterval == 0 {
		return
	}
	endBlock := new(big.Int).Add(saigonBlock, new(big.Int).SetUint64(config.SaigonFundInterval*(config.SaigonFundRepeat-1))) // additional token will be minted at block 0 of each interval 4 intervals
	if headBlock.Cmp(saigonBlock) < 0 || headBlock.Cmp(endBlock) > 0 {
		return
	}
	blockOfInterval := new(big.Int).Mod(new(big.Int).Sub(headBlock, saigonBlock), new(big.Int).SetUint64(config.SaigonFundInterval))
	if blockOfInterval.Cmp(big.NewInt(0)) == 0 {
		if config.SaigonFundAmount != nil {
			ecoSystemFund := (*big.Int)(config.SaigonFundAmount)
			statedb.AddBalance(config.SaigonFundAddress, ecoSystemFund)
		}
	}
}

func ApplyVIPVRC25Upgarde(statedb *state.StateDB, config *params.VictionConfig, atlasBlock *big.Int, headBlock *big.Int) {
	if headBlock.Cmp(atlasBlock) == 0 {
		if config.AtlasVRC25MinCap != nil {
			slotHash := common.BigToHash(new(big.Int).SetUint64(vrc25.SlotVRC25Contract["minCap"]))
			statedb.SetState(config.VRC25Contract, slotHash, common.BigToHash((*big.Int)(config.AtlasVRC25MinCap)))
		}
	}
}	
