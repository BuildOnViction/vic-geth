// Copyright 2016 The go-ethereum Authors
// (original work)
// Copyright 2025 The Viction Authors
// (modifications)
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
)

type VictionConfig struct {
	AtlasVRC25MinCap *math.Decimal256 `json:"atlasVRC25MinCap,omitempty"`

	ConsensusLegacyCompat      bool   `json:"consensusLegacyCompat,omitempty"`
	ConsensusLimitTimeFinality uint64 `json:"consensusLimitTimeFinality,omitempty"`

	LendingContract             common.Address   `json:"lendingContract,omitempty"`
	LendingFinalizedContract    common.Address   `json:"lendingFinalizedContract,omitempty"`
	LendingInterestBase         *math.Decimal256 `json:"lendingInterestBase,omitempty"`
	LendingLiquidateTradeBlock  uint64           `json:"lendingLiquidateTradeBlock,omitempty"`
	LendingRecallBase           *math.Decimal256 `json:"lendingRecallBase,omitempty"`
	LendingRegistrationContract common.Address   `json:"lendingRegistrationContract,omitempty"`
	LendingTopupBase            uint64           `json:"lendingTopupBase,omitempty"`
	LendingTopupRate            uint64           `json:"lendingTopupRate,omitempty"`

	PenaltyComebackBlockCount uint64 `json:"penaltyComebackBlockCount,omitempty"`
	PenaltyEpochCount         uint64 `json:"penaltyEpochCount,omitempty"`

	RandomizerCommitNthBlock uint64         `json:"randomizerCommitNthBlock,omitempty"`
	RandomizerContract       common.Address `json:"randomizerContract,omitempty"`
	RandomizerFinaleNthBlock uint64         `json:"randomizerFinaleNthBlock,omitempty"`
	RandomizerRevealNthBlock uint64         `json:"randomizerRevealNthBlock,omitempty"`

	RelayerLendingCancelFee     *math.Decimal256 `json:"relayerLendingCancelFee,omitempty"`
	RelayerLendingFee           *math.Decimal256 `json:"relayerLendingFee,omitempty"`
	RelayerLockedAddress        common.Address   `json:"relayerLockedAddress,omitempty"`
	RelayerLockedAmount         *math.Decimal256 `json:"relayerLockedAmount,omitempty"`
	RelayerRegistrationContract common.Address   `json:"relayerRegistrationContract,omitempty"`
	RelayerTradingCancelFee     *math.Decimal256 `json:"relayerTradingCancelFee,omitempty"`
	RelayerTradingFee           *math.Decimal256 `json:"relayerTradingFee,omitempty"`

	RewardFoundationAddress common.Address   `json:"rewardFoundationAddress,omitempty"`
	RewardFoundationPercent uint64           `json:"rewardFoundationPercent,omitempty"`
	RewardPerEpoch          *math.Decimal256 `json:"rewardPerEpoch,omitempty"`
	RewardValidatorPercent  uint64           `json:"rewardValidatorPercent,omitempty"`
	RewardVoterPercent      uint64           `json:"rewardVoterPercent,omitempty"`

	TRC21GasPrice    *math.Decimal256 `json:"trc21GasPrice,omitempty"`
	TRC21NewGasPrice *math.Decimal256 `json:"trc21NewGasPrice,omitempty"`

	SaigonFundAddress    common.Address   `json:"saigonFundAddress,omitempty"`
	SaigonFundAmount     *math.Decimal256 `json:"saigonFundAmount,omitempty"`
	SaigonFundInterval   uint64           `json:"saigonFundInterval,omitempty"`
	SaigonFundRepeat     uint64           `json:"saigonFundRepeat,omitempty"`
	SaigonRewardPerEpoch *math.Decimal256 `json:"saigonRewardPerEpoch,omitempty"`

	TradingBaseCancelFee   *math.Decimal256 `json:"tradingBaseCancelFee,omitempty"`
	TradingBaseFee         *math.Decimal256 `json:"tradingBaseFee,omitempty"`
	TradingBasePrice       *math.Decimal256 `json:"tradingBasePrice,omitempty"`
	TradingContract        common.Address   `json:"tradingContract,omitempty"`
	TradingListingContract common.Address   `json:"tradingListingContract,omitempty"`
	TradingStateContract   common.Address   `json:"tradingStateContract,omitempty"`

	ValidatorBlockSignContract     common.Address `json:"validatorBlockSignContract,omitempty"`
	ValidatorContract              common.Address `json:"validatorContract,omitempty"`
	ValidatorMaxCount              uint64         `json:"validatorMaxCount,omitempty"`
	ValidatorMinBlockPerEpochCount uint64         `json:"validatorMinBlockPerEpochCount,omitempty"`
	ValidatorSignInterval          uint64         `json:"validatorSignInterval,omitempty"`

	VRC25GasPrice         *math.Decimal256 `json:"vrc25GasPrice,omitempty"`
	VRC25RegistryContract common.Address   `json:"vrc25RegistryContract,omitempty"`
}

// Enable bypass balance check for given block number and addr.
func (c *VictionConfig) GetBypassBalance(number uint64, addr common.Address) *big.Int {
	if c == nil || !c.ConsensusLegacyCompat {
		return nil
	}
	if fixAddr, ok := victionBalanceFixBlocks[number]; ok {
		if fixAddr == addr {
			if balanceStr, ok := victionBalanceFixAmounts[fixAddr]; ok {
				val := new(big.Int)
				val.SetString(balanceStr+"000000000000000000", 10)
				return val
			}
		}
	}
	return nil
}

// Check if the given address is blocked from interacting with the network.
func (c *VictionConfig) IsBlacklisted(addr common.Address, number *big.Int) bool {
	if c == nil || !c.ConsensusLegacyCompat {
		return false
	}
	if blocked, exists := victionBlacklists[addr]; exists {
		if number == nil {
			return true
		}
		return number.Uint64() >= blocked
	}
	return false
}

// Enable bypass validator signature for given block number.
func (c *VictionConfig) IsBypassValidatorBlock(number uint64) bool {
	if c == nil || !c.ConsensusLegacyCompat {
		return false
	}
	return victionBypassValidators[number]
}

// Check if current blockOfEpoch is in secret committing phase.
func (c *VictionConfig) IsRandomizerCommitPhase(blockOfEpoch uint64) bool {
	return blockOfEpoch > 0 && blockOfEpoch >= c.RandomizerCommitNthBlock && blockOfEpoch < c.RandomizerRevealNthBlock
}

// Check if current blockOfEpoch is in secret revealing phase.
func (c *VictionConfig) IsRandomizerOpeningPhase(blockOfEpoch uint64) bool {
	return blockOfEpoch > 0 && blockOfEpoch >= c.RandomizerRevealNthBlock && blockOfEpoch <= c.RandomizerFinaleNthBlock
}

// Check if PoSV configuration is available.
func (c *ChainConfig) IsPosv() bool {
	return c.Posv != nil
}

// Check if Viction configuration is available.
func (c *ChainConfig) IsViction() bool {
	return c.Viction != nil
}

// Viction Mainnet only
var victionBlacklists = map[common.Address]uint64{
	common.HexToAddress("0x5248bfb72fd4f234e062d3e9bb76f08643004fcd"): 9349100,
	common.HexToAddress("0x5ac26105b35ea8935be382863a70281ec7a985e9"): 9349100,
	common.HexToAddress("0x09c4f991a41e7ca0645d7dfbfee160b55e562ea4"): 9349100,
	common.HexToAddress("0xb3157bbc5b401a45d6f60b106728bb82ebaa585b"): 9349100,
	common.HexToAddress("0x741277a8952128d5c2ffe0550f5001e4c8247674"): 9349100,
	common.HexToAddress("0x10ba49c1caa97d74b22b3e74493032b180cebe01"): 9349100,
	common.HexToAddress("0x07048d51d9e6179578a6e3b9ee28cdc183b865e4"): 9349100,
	common.HexToAddress("0x4b899001d73c7b4ec404a771d37d9be13b8983de"): 9349100,
	common.HexToAddress("0x85cb320a9007f26b7652c19a2a65db1da2d0016f"): 9349100,
	common.HexToAddress("0x06869dbd0e3a2ea37ddef832e20fa005c6f0ca39"): 9349100,
	common.HexToAddress("0x82e48bc7e2c93d89125428578fb405947764ad7c"): 9349100,
	common.HexToAddress("0x1f9a78534d61732367cbb43fc6c89266af67c989"): 9349100,
	common.HexToAddress("0x7c3b1fa91df55ff7af0cad9e0399384dc5c6641b"): 9349100,
	common.HexToAddress("0x5888dc1ceb0ff632713486b9418e59743af0fd20"): 9349100,
	common.HexToAddress("0xa512fa1c735fc3cc635624d591dd9ea1ce339ca5"): 9349100,
	common.HexToAddress("0x0832517654c7b7e36b1ef45d76de70326b09e2c7"): 9349100,
	common.HexToAddress("0xca14e3c4c78bafb60819a78ff6e6f0f709d2aea7"): 9349100,
	common.HexToAddress("0x652ce195a23035114849f7642b0e06647d13e57a"): 9349100,
	common.HexToAddress("0x29a79f00f16900999d61b6e171e44596af4fb5ae"): 9349100,
	common.HexToAddress("0xf9fd1c2b0af0d91b0b6754e55639e3f8478dd04a"): 9349100,
	common.HexToAddress("0xb835710c9901d5fe940ef1b99ed918902e293e35"): 9349100,
	common.HexToAddress("0x04dd29ce5c253377a9a3796103ea0d9a9e514153"): 9349100,
	common.HexToAddress("0x2b4b56846eaf05c1fd762b5e1ac802efd0ab871c"): 9349100,
	common.HexToAddress("0x1d1f909f6600b23ce05004f5500ab98564717996"): 9349100,
	common.HexToAddress("0x0dfdcebf80006dc9ab7aae8c216b51c6b6759e86"): 9349100,
	common.HexToAddress("0x2b373890a28e5e46197fbc04f303bbfdd344056f"): 9349100,
	common.HexToAddress("0xa8a3ef3dc5d8e36aee76f3671ec501ec31e28254"): 9349100,
	common.HexToAddress("0x4f3d18136fe2b5665c29bdaf74591fc6625ef427"): 9349100,
	common.HexToAddress("0x175d728b0e0f1facb5822a2e0c03bde93596e324"): 9349100,
	common.HexToAddress("0xd575c2611984fcd79513b80ab94f59dc5bab4916"): 9349100,
	common.HexToAddress("0x0579337873c97c4ba051310236ea847f5be41bc0"): 9349100,
	common.HexToAddress("0xed12a519cc15b286920fc15fd86106b3e6a16218"): 9349100,
	common.HexToAddress("0x492d26d852a0a0a2982bb40ec86fe394488c419e"): 9349100,
	common.HexToAddress("0xce5c7635d02dc4e1d6b46c256cae6323be294a32"): 9349100,
	common.HexToAddress("0x8b94db158b5e78a6c032c7e7c9423dec62c8b11c"): 9349100,
	common.HexToAddress("0x0e7c48c085b6b0aa7ca6e4cbcc8b9a92dc270eb4"): 9349100,
	common.HexToAddress("0x206e6508462033ef8425edc6c10789d241d49acb"): 9349100,
	common.HexToAddress("0x7710e7b7682f26cb5a1202e1cff094fbf7777758"): 9349100,
	common.HexToAddress("0xcb06f949313b46bbf53b8e6b2868a0c260ff9385"): 9349100,
	common.HexToAddress("0xf884e43533f61dc2997c0e19a6eff33481920c00"): 9349100,
	common.HexToAddress("0x8b635ef2e4c8fe21fc2bda027eb5f371d6aa2fc1"): 9349100,
	common.HexToAddress("0x10f01a27cf9b29d02ce53497312b96037357a361"): 9349100,
	common.HexToAddress("0x693dd49b0ed70f162d733cf20b6c43dc2a2b4d95"): 9349100,
	common.HexToAddress("0xe0bec72d1c2a7a7fb0532cdfac44ebab9f6f41ee"): 9349100,
	common.HexToAddress("0xc8793633a537938cb49cdbbffd45428f10e45b64"): 9349100,
	common.HexToAddress("0x0d07a6cbbe9fa5c4f154e5623bfe47fb4d857d8e"): 9349100,
	common.HexToAddress("0xd4080b289da95f70a586610c38268d8d4cf1e4c4"): 9349100,
	common.HexToAddress("0x8bcfb0caf41f0aa1b548cae76dcdd02e33866a1b"): 9349100,
	common.HexToAddress("0xabfef22b92366d3074676e77ea911ccaabfb64c1"): 9349100,
	common.HexToAddress("0xcc4df7a32faf3efba32c9688def5ccf9fefe443d"): 9349100,
	common.HexToAddress("0x7ec1e48a582475f5f2b7448a86c4ea7a26ea36f8"): 9349100,
	common.HexToAddress("0xe3de67289080f63b0c2612844256a25bb99ac0ad"): 9349100,
	common.HexToAddress("0x3ba623300cf9e48729039b3c9e0dee9b785d636e"): 9349100,
	common.HexToAddress("0x402f2cfc9c8942f5e7a12c70c625d07a5d52fe29"): 9349100,
	common.HexToAddress("0xd62358d42afbde095a4ca868581d85f9adcc3d61"): 9349100,
	common.HexToAddress("0x3969f86acb733526cd61e3c6e3b4660589f32bc6"): 9349100,
	common.HexToAddress("0x67615413d7cdadb2c435a946aec713a9a9794d39"): 9349100,
	common.HexToAddress("0xfe685f43acc62f92ab01a8da80d76455d39d3cb3"): 9349100,
	common.HexToAddress("0x3538a544021c07869c16b764424c5987409cba48"): 9349100,
	common.HexToAddress("0xe187cf86c2274b1f16e8225a7da9a75aba4f1f5f"): 9349100,
}

// Viction Mainnet only
var victionBalanceFixAmounts = map[common.Address]string{
	common.HexToAddress("0x5248bfb72fd4f234e062d3e9bb76f08643004fcd"): "29410",
	common.HexToAddress("0x5ac26105b35ea8935be382863a70281ec7a985e9"): "23551",
	common.HexToAddress("0x09c4f991a41e7ca0645d7dfbfee160b55e562ea4"): "25821",
	common.HexToAddress("0xb3157bbc5b401a45d6f60b106728bb82ebaa585b"): "20051",
	common.HexToAddress("0x741277a8952128d5c2ffe0550f5001e4c8247674"): "23937",
	common.HexToAddress("0x10ba49c1caa97d74b22b3e74493032b180cebe01"): "27320",
	common.HexToAddress("0x07048d51d9e6179578a6e3b9ee28cdc183b865e4"): "29758",
	common.HexToAddress("0x4b899001d73c7b4ec404a771d37d9be13b8983de"): "26148",
	common.HexToAddress("0x85cb320a9007f26b7652c19a2a65db1da2d0016f"): "27216",
	common.HexToAddress("0x06869dbd0e3a2ea37ddef832e20fa005c6f0ca39"): "29449",
	common.HexToAddress("0x82e48bc7e2c93d89125428578fb405947764ad7c"): "28084",
	common.HexToAddress("0x1f9a78534d61732367cbb43fc6c89266af67c989"): "29287",
	common.HexToAddress("0x7c3b1fa91df55ff7af0cad9e0399384dc5c6641b"): "21574",
	common.HexToAddress("0x5888dc1ceb0ff632713486b9418e59743af0fd20"): "28836",
	common.HexToAddress("0xa512fa1c735fc3cc635624d591dd9ea1ce339ca5"): "25515",
	common.HexToAddress("0x0832517654c7b7e36b1ef45d76de70326b09e2c7"): "22873",
	common.HexToAddress("0xca14e3c4c78bafb60819a78ff6e6f0f709d2aea7"): "24968",
	common.HexToAddress("0x652ce195a23035114849f7642b0e06647d13e57a"): "24091",
	common.HexToAddress("0x29a79f00f16900999d61b6e171e44596af4fb5ae"): "20790",
	common.HexToAddress("0xf9fd1c2b0af0d91b0b6754e55639e3f8478dd04a"): "23331",
	common.HexToAddress("0xb835710c9901d5fe940ef1b99ed918902e293e35"): "28273",
	common.HexToAddress("0x04dd29ce5c253377a9a3796103ea0d9a9e514153"): "29956",
	common.HexToAddress("0x2b4b56846eaf05c1fd762b5e1ac802efd0ab871c"): "24911",
	common.HexToAddress("0x1d1f909f6600b23ce05004f5500ab98564717996"): "25637",
	common.HexToAddress("0x0dfdcebf80006dc9ab7aae8c216b51c6b6759e86"): "26378",
	common.HexToAddress("0x2b373890a28e5e46197fbc04f303bbfdd344056f"): "21109",
	common.HexToAddress("0xa8a3ef3dc5d8e36aee76f3671ec501ec31e28254"): "22072",
	common.HexToAddress("0x4f3d18136fe2b5665c29bdaf74591fc6625ef427"): "21650",
	common.HexToAddress("0x175d728b0e0f1facb5822a2e0c03bde93596e324"): "21588",
	common.HexToAddress("0xd575c2611984fcd79513b80ab94f59dc5bab4916"): "28971",
	common.HexToAddress("0x0579337873c97c4ba051310236ea847f5be41bc0"): "28344",
	common.HexToAddress("0xed12a519cc15b286920fc15fd86106b3e6a16218"): "24443",
	common.HexToAddress("0x492d26d852a0a0a2982bb40ec86fe394488c419e"): "26623",
	common.HexToAddress("0xce5c7635d02dc4e1d6b46c256cae6323be294a32"): "28459",
	common.HexToAddress("0x8b94db158b5e78a6c032c7e7c9423dec62c8b11c"): "21803",
	common.HexToAddress("0x0e7c48c085b6b0aa7ca6e4cbcc8b9a92dc270eb4"): "21739",
	common.HexToAddress("0x206e6508462033ef8425edc6c10789d241d49acb"): "21883",
	common.HexToAddress("0x7710e7b7682f26cb5a1202e1cff094fbf7777758"): "28907",
	common.HexToAddress("0xcb06f949313b46bbf53b8e6b2868a0c260ff9385"): "28932",
	common.HexToAddress("0xf884e43533f61dc2997c0e19a6eff33481920c00"): "27780",
	common.HexToAddress("0x8b635ef2e4c8fe21fc2bda027eb5f371d6aa2fc1"): "23115",
	common.HexToAddress("0x10f01a27cf9b29d02ce53497312b96037357a361"): "22716",
	common.HexToAddress("0x693dd49b0ed70f162d733cf20b6c43dc2a2b4d95"): "20020",
	common.HexToAddress("0xe0bec72d1c2a7a7fb0532cdfac44ebab9f6f41ee"): "23071",
	common.HexToAddress("0xc8793633a537938cb49cdbbffd45428f10e45b64"): "24652",
	common.HexToAddress("0x0d07a6cbbe9fa5c4f154e5623bfe47fb4d857d8e"): "21907",
	common.HexToAddress("0xd4080b289da95f70a586610c38268d8d4cf1e4c4"): "22719",
	common.HexToAddress("0x8bcfb0caf41f0aa1b548cae76dcdd02e33866a1b"): "29062",
	common.HexToAddress("0xabfef22b92366d3074676e77ea911ccaabfb64c1"): "23110",
	common.HexToAddress("0xcc4df7a32faf3efba32c9688def5ccf9fefe443d"): "21397",
	common.HexToAddress("0x7ec1e48a582475f5f2b7448a86c4ea7a26ea36f8"): "23105",
	common.HexToAddress("0xe3de67289080f63b0c2612844256a25bb99ac0ad"): "29721",
	common.HexToAddress("0x3ba623300cf9e48729039b3c9e0dee9b785d636e"): "25917",
	common.HexToAddress("0x402f2cfc9c8942f5e7a12c70c625d07a5d52fe29"): "24712",
	common.HexToAddress("0xd62358d42afbde095a4ca868581d85f9adcc3d61"): "24449",
	common.HexToAddress("0x3969f86acb733526cd61e3c6e3b4660589f32bc6"): "29579",
	common.HexToAddress("0x67615413d7cdadb2c435a946aec713a9a9794d39"): "26333",
	common.HexToAddress("0xfe685f43acc62f92ab01a8da80d76455d39d3cb3"): "29825",
	common.HexToAddress("0x3538a544021c07869c16b764424c5987409cba48"): "22746",
	common.HexToAddress("0xe187cf86c2274b1f16e8225a7da9a75aba4f1f5f"): "23734",
}

// Viction Mainnet only
var victionBalanceFixBlocks = map[uint64]common.Address{
	9073579: common.HexToAddress("0x5248bfb72fd4f234e062d3e9bb76f08643004fcd"),
	9147130: common.HexToAddress("0x5ac26105b35ea8935be382863a70281ec7a985e9"),
	9147195: common.HexToAddress("0x09c4f991a41e7ca0645d7dfbfee160b55e562ea4"),
	9147200: common.HexToAddress("0xb3157bbc5b401a45d6f60b106728bb82ebaa585b"),
	9147206: common.HexToAddress("0x741277a8952128d5c2ffe0550f5001e4c8247674"),
	9147212: common.HexToAddress("0x10ba49c1caa97d74b22b3e74493032b180cebe01"),
	9147217: common.HexToAddress("0x07048d51d9e6179578a6e3b9ee28cdc183b865e4"),
	9147223: common.HexToAddress("0x4b899001d73c7b4ec404a771d37d9be13b8983de"),
	9147229: common.HexToAddress("0x85cb320a9007f26b7652c19a2a65db1da2d0016f"),
	9147234: common.HexToAddress("0x06869dbd0e3a2ea37ddef832e20fa005c6f0ca39"),
	9147240: common.HexToAddress("0x82e48bc7e2c93d89125428578fb405947764ad7c"),
	9147246: common.HexToAddress("0x1f9a78534d61732367cbb43fc6c89266af67c989"),
	9147251: common.HexToAddress("0x7c3b1fa91df55ff7af0cad9e0399384dc5c6641b"),
	9147257: common.HexToAddress("0x5888dc1ceb0ff632713486b9418e59743af0fd20"),
	9147263: common.HexToAddress("0xa512fa1c735fc3cc635624d591dd9ea1ce339ca5"),
	9147268: common.HexToAddress("0x0832517654c7b7e36b1ef45d76de70326b09e2c7"),
	9147274: common.HexToAddress("0xca14e3c4c78bafb60819a78ff6e6f0f709d2aea7"),
	9147279: common.HexToAddress("0x652ce195a23035114849f7642b0e06647d13e57a"),
	9147285: common.HexToAddress("0x29a79f00f16900999d61b6e171e44596af4fb5ae"),
	9147291: common.HexToAddress("0xf9fd1c2b0af0d91b0b6754e55639e3f8478dd04a"),
	9147296: common.HexToAddress("0xb835710c9901d5fe940ef1b99ed918902e293e35"),
	9147302: common.HexToAddress("0x04dd29ce5c253377a9a3796103ea0d9a9e514153"),
	9147308: common.HexToAddress("0x2b4b56846eaf05c1fd762b5e1ac802efd0ab871c"),
	9147314: common.HexToAddress("0x1d1f909f6600b23ce05004f5500ab98564717996"),
	9147319: common.HexToAddress("0x0dfdcebf80006dc9ab7aae8c216b51c6b6759e86"),
	9147325: common.HexToAddress("0x2b373890a28e5e46197fbc04f303bbfdd344056f"),
	9147330: common.HexToAddress("0xa8a3ef3dc5d8e36aee76f3671ec501ec31e28254"),
	9147336: common.HexToAddress("0x4f3d18136fe2b5665c29bdaf74591fc6625ef427"),
	9147342: common.HexToAddress("0x175d728b0e0f1facb5822a2e0c03bde93596e324"),
	9145281: common.HexToAddress("0xd575c2611984fcd79513b80ab94f59dc5bab4916"),
	9145315: common.HexToAddress("0x0579337873c97c4ba051310236ea847f5be41bc0"),
	9145341: common.HexToAddress("0xed12a519cc15b286920fc15fd86106b3e6a16218"),
	9145367: common.HexToAddress("0x492d26d852a0a0a2982bb40ec86fe394488c419e"),
	9145386: common.HexToAddress("0xce5c7635d02dc4e1d6b46c256cae6323be294a32"),
	9145414: common.HexToAddress("0x8b94db158b5e78a6c032c7e7c9423dec62c8b11c"),
	9145436: common.HexToAddress("0x0e7c48c085b6b0aa7ca6e4cbcc8b9a92dc270eb4"),
	9145463: common.HexToAddress("0x206e6508462033ef8425edc6c10789d241d49acb"),
	9145493: common.HexToAddress("0x7710e7b7682f26cb5a1202e1cff094fbf7777758"),
	9145519: common.HexToAddress("0xcb06f949313b46bbf53b8e6b2868a0c260ff9385"),
	9145549: common.HexToAddress("0xf884e43533f61dc2997c0e19a6eff33481920c00"),
	9147352: common.HexToAddress("0x8b635ef2e4c8fe21fc2bda027eb5f371d6aa2fc1"),
	9147357: common.HexToAddress("0x10f01a27cf9b29d02ce53497312b96037357a361"),
	9147363: common.HexToAddress("0x693dd49b0ed70f162d733cf20b6c43dc2a2b4d95"),
	9147369: common.HexToAddress("0xe0bec72d1c2a7a7fb0532cdfac44ebab9f6f41ee"),
	9147375: common.HexToAddress("0xc8793633a537938cb49cdbbffd45428f10e45b64"),
	9147380: common.HexToAddress("0x0d07a6cbbe9fa5c4f154e5623bfe47fb4d857d8e"),
	9147386: common.HexToAddress("0xd4080b289da95f70a586610c38268d8d4cf1e4c4"),
	9147392: common.HexToAddress("0x8bcfb0caf41f0aa1b548cae76dcdd02e33866a1b"),
	9147397: common.HexToAddress("0xabfef22b92366d3074676e77ea911ccaabfb64c1"),
	9147403: common.HexToAddress("0xcc4df7a32faf3efba32c9688def5ccf9fefe443d"),
	9147408: common.HexToAddress("0x7ec1e48a582475f5f2b7448a86c4ea7a26ea36f8"),
	9147414: common.HexToAddress("0xe3de67289080f63b0c2612844256a25bb99ac0ad"),
	9147420: common.HexToAddress("0x3ba623300cf9e48729039b3c9e0dee9b785d636e"),
	9147425: common.HexToAddress("0x402f2cfc9c8942f5e7a12c70c625d07a5d52fe29"),
	9147431: common.HexToAddress("0xd62358d42afbde095a4ca868581d85f9adcc3d61"),
	9147437: common.HexToAddress("0x3969f86acb733526cd61e3c6e3b4660589f32bc6"),
	9147442: common.HexToAddress("0x67615413d7cdadb2c435a946aec713a9a9794d39"),
	9147448: common.HexToAddress("0xfe685f43acc62f92ab01a8da80d76455d39d3cb3"),
	9147453: common.HexToAddress("0x3538a544021c07869c16b764424c5987409cba48"),
	9147459: common.HexToAddress("0xe187cf86c2274b1f16e8225a7da9a75aba4f1f5f"),
}

// Viction Mainnet only
var victionBypassValidators = map[uint64]bool{
	14458500: true,
}

var victionHardforks = map[string]bool{
	"tip2019Block":          true,
	"tipSigningBlock":       true,
	"tipRandomizeBlock":     true,
	"tipGasPriceBlock":      true,
	"tipNativeTradingBlock": true,
	"tipNativeLendingBlock": true,
	"tip2021Block":          true,

	"saigonBlock":       true,
	"atlasBlock":        true,
	"atlasRefreshBlock": true,
	"prometheusBlock":   true,
}

func isVictionHardfork(name string) bool {
	if _, ok := victionHardforks[name]; ok {
		return true
	}
	return false
}
