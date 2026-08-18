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
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

// Genesis hashes to enforce below configs on.
var (
	MainnetGenesisHash   = common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3")
	VictionGenesisHash   = common.HexToHash("0x9326145f8a2c8c00bbe13afc7d7f3d9c868b5ef39d89f2f4e9390e9720298624")
	RopstenGenesisHash   = common.HexToHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d")
	RinkebyGenesisHash   = common.HexToHash("0x6341fd3daf94b748c72ced5a5b26028f2474f5f00d824504e4fa37a75767e177")
	GoerliGenesisHash    = common.HexToHash("0xbf7e331f7f7c1dd2e05159666b3bf8bc7a8a3a9eb1d518969eab529dd9b88c1a")
	CalaverasGenesisHash = common.HexToHash("0xeb9233d066c275efcdfed8037f4fc082770176aefdbcb7691c71da412a5670f2")
	VictestGenesisHash   = common.HexToHash("0x296f14cfe39dd2ce9cd2dcf2bd5973c9b59531bc239e7d445c66268b172e52e3")
)

// TrustedCheckpoints associates each known checkpoint with the genesis hash of
// the chain it belongs to.
var TrustedCheckpoints = map[common.Hash]*TrustedCheckpoint{
	MainnetGenesisHash: MainnetTrustedCheckpoint,
	RopstenGenesisHash: RopstenTrustedCheckpoint,
	RinkebyGenesisHash: RinkebyTrustedCheckpoint,
	GoerliGenesisHash:  GoerliTrustedCheckpoint,
}

// CheckpointOracles associates each known checkpoint oracles with the genesis hash of
// the chain it belongs to.
var CheckpointOracles = map[common.Hash]*CheckpointOracleConfig{
	MainnetGenesisHash: MainnetCheckpointOracle,
	RopstenGenesisHash: RopstenCheckpointOracle,
	RinkebyGenesisHash: RinkebyCheckpointOracle,
	GoerliGenesisHash:  GoerliCheckpointOracle,
}

var (
	// MainnetChainConfig is the chain parameters to run a node on the main network.
	MainnetChainConfig = &ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(1_150_000),
		DAOForkBlock:        big.NewInt(1_920_000),
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(2_463_000),
		EIP150Hash:          common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0"),
		EIP155Block:         big.NewInt(2_675_000),
		EIP158Block:         big.NewInt(2_675_000),
		ByzantiumBlock:      big.NewInt(4_370_000),
		ConstantinopleBlock: big.NewInt(7_280_000),
		PetersburgBlock:     big.NewInt(7_280_000),
		IstanbulBlock:       big.NewInt(9_069_000),
		MuirGlacierBlock:    big.NewInt(9_200_000),
		BerlinBlock:         big.NewInt(12_244_000),
		LondonBlock:         big.NewInt(12_965_000),
		Ethash:              new(EthashConfig),
	}

	// VictionChainConfig is the chain parameters to run a Viction Mainnet node.
	VictionChainConfig = readChainSpec("chainspecs/viction.json")

	// MainnetTrustedCheckpoint contains the light client trusted checkpoint for the main network.
	MainnetTrustedCheckpoint = &TrustedCheckpoint{
		SectionIndex: 395,
		SectionHead:  common.HexToHash("0xbfca95b8c1de014e252288e9c32029825fadbff58285f5b54556525e480dbb5b"),
		CHTRoot:      common.HexToHash("0x2ccf3dbb58eb6375e037fdd981ca5778359e4b8fa0270c2878b14361e64161e7"),
		BloomRoot:    common.HexToHash("0x2d46ec65a6941a2dc1e682f8f81f3d24192021f492fdf6ef0fdd51acb0f4ba0f"),
	}

	// MainnetCheckpointOracle contains a set of configs for the main network oracle.
	MainnetCheckpointOracle = &CheckpointOracleConfig{
		Address: common.HexToAddress("0x9a9070028361F7AAbeB3f2F2Dc07F82C4a98A02a"),
		Signers: []common.Address{
			common.HexToAddress("0x1b2C260efc720BE89101890E4Db589b44E950527"), // Peter
			common.HexToAddress("0x78d1aD571A1A09D60D9BBf25894b44e4C8859595"), // Martin
			common.HexToAddress("0x286834935f4A8Cfb4FF4C77D5770C2775aE2b0E7"), // Zsolt
			common.HexToAddress("0xb86e2B0Ab5A4B1373e40c51A7C712c70Ba2f9f8E"), // Gary
			common.HexToAddress("0x0DF8fa387C602AE62559cC4aFa4972A7045d6707"), // Guillaume
		},
		Threshold: 2,
	}

	// RopstenChainConfig contains the chain parameters to run a node on the Ropsten test network.
	RopstenChainConfig = &ChainConfig{
		ChainID:             big.NewInt(3),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.HexToHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d"),
		EIP155Block:         big.NewInt(10),
		EIP158Block:         big.NewInt(10),
		ByzantiumBlock:      big.NewInt(1_700_000),
		ConstantinopleBlock: big.NewInt(4_230_000),
		PetersburgBlock:     big.NewInt(4_939_394),
		IstanbulBlock:       big.NewInt(6_485_846),
		MuirGlacierBlock:    big.NewInt(7_117_117),
		BerlinBlock:         big.NewInt(9_812_189),
		LondonBlock:         big.NewInt(10_499_401),
		Ethash:              new(EthashConfig),
	}

	// RopstenTrustedCheckpoint contains the light client trusted checkpoint for the Ropsten test network.
	RopstenTrustedCheckpoint = &TrustedCheckpoint{
		SectionIndex: 329,
		SectionHead:  common.HexToHash("0xe66f7038333a01fb95dc9ea03e5a2bdaf4b833cdcb9e393b9127e013bd64d39b"),
		CHTRoot:      common.HexToHash("0x1b0c883338ac0d032122800c155a2e73105fbfebfaa50436893282bc2d9feec5"),
		BloomRoot:    common.HexToHash("0x3cc98c88d283bf002378246f22c653007655cbcea6ed89f98d739f73bd341a01"),
	}

	// RopstenCheckpointOracle contains a set of configs for the Ropsten test network oracle.
	RopstenCheckpointOracle = &CheckpointOracleConfig{
		Address: common.HexToAddress("0xEF79475013f154E6A65b54cB2742867791bf0B84"),
		Signers: []common.Address{
			common.HexToAddress("0x32162F3581E88a5f62e8A61892B42C46E2c18f7b"), // Peter
			common.HexToAddress("0x78d1aD571A1A09D60D9BBf25894b44e4C8859595"), // Martin
			common.HexToAddress("0x286834935f4A8Cfb4FF4C77D5770C2775aE2b0E7"), // Zsolt
			common.HexToAddress("0xb86e2B0Ab5A4B1373e40c51A7C712c70Ba2f9f8E"), // Gary
			common.HexToAddress("0x0DF8fa387C602AE62559cC4aFa4972A7045d6707"), // Guillaume
		},
		Threshold: 2,
	}

	// RinkebyChainConfig contains the chain parameters to run a node on the Rinkeby test network.
	RinkebyChainConfig = &ChainConfig{
		ChainID:             big.NewInt(4),
		HomesteadBlock:      big.NewInt(1),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(2),
		EIP150Hash:          common.HexToHash("0x9b095b36c15eaf13044373aef8ee0bd3a382a5abb92e402afa44b8249c3a90e9"),
		EIP155Block:         big.NewInt(3),
		EIP158Block:         big.NewInt(3),
		ByzantiumBlock:      big.NewInt(1_035_301),
		ConstantinopleBlock: big.NewInt(3_660_663),
		PetersburgBlock:     big.NewInt(4_321_234),
		IstanbulBlock:       big.NewInt(5_435_345),
		MuirGlacierBlock:    nil,
		BerlinBlock:         big.NewInt(8_290_928),
		LondonBlock:         big.NewInt(8_897_988),
		Clique: &CliqueConfig{
			Period: 15,
			Epoch:  30000,
		},
	}

	// RinkebyTrustedCheckpoint contains the light client trusted checkpoint for the Rinkeby test network.
	RinkebyTrustedCheckpoint = &TrustedCheckpoint{
		SectionIndex: 276,
		SectionHead:  common.HexToHash("0xea89a4b04e3da9bd688e316f8de669396b6d4a38a19d2cd96a00b70d58b836aa"),
		CHTRoot:      common.HexToHash("0xd6889d0bf6673c0d2c1cf6e9098a6fe5b30888a115b6112796aa8ee8efc4a723"),
		BloomRoot:    common.HexToHash("0x6009a9256b34b8bde3a3f094afb647ba5d73237546017b9025d64ac1ff54c47c"),
	}

	// RinkebyCheckpointOracle contains a set of configs for the Rinkeby test network oracle.
	RinkebyCheckpointOracle = &CheckpointOracleConfig{
		Address: common.HexToAddress("0xebe8eFA441B9302A0d7eaECc277c09d20D684540"),
		Signers: []common.Address{
			common.HexToAddress("0xd9c9cd5f6779558b6e0ed4e6acf6b1947e7fa1f3"), // Peter
			common.HexToAddress("0x78d1aD571A1A09D60D9BBf25894b44e4C8859595"), // Martin
			common.HexToAddress("0x286834935f4A8Cfb4FF4C77D5770C2775aE2b0E7"), // Zsolt
			common.HexToAddress("0xb86e2B0Ab5A4B1373e40c51A7C712c70Ba2f9f8E"), // Gary
		},
		Threshold: 2,
	}

	// GoerliChainConfig contains the chain parameters to run a node on the Görli test network.
	GoerliChainConfig = &ChainConfig{
		ChainID:             big.NewInt(5),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(1_561_651),
		MuirGlacierBlock:    nil,
		BerlinBlock:         big.NewInt(4_460_644),
		LondonBlock:         big.NewInt(5_062_605),
		Clique: &CliqueConfig{
			Period: 15,
			Epoch:  30000,
		},
	}

	// GoerliTrustedCheckpoint contains the light client trusted checkpoint for the Görli test network.
	GoerliTrustedCheckpoint = &TrustedCheckpoint{
		SectionIndex: 160,
		SectionHead:  common.HexToHash("0xb5a666c790dc35a5613d04ebba8ba47a850b45a15d9b95ad7745c35ae034b5a5"),
		CHTRoot:      common.HexToHash("0x6b4e00df52bdc38fa6c26c8ef595c2ad6184963ea36ab08ee744af460aa735e1"),
		BloomRoot:    common.HexToHash("0x8fa88f5e50190cb25243aeee262a1a9e4434a06f8d455885dcc1b5fc48c33836"),
	}

	// GoerliCheckpointOracle contains a set of configs for the Goerli test network oracle.
	GoerliCheckpointOracle = &CheckpointOracleConfig{
		Address: common.HexToAddress("0x18CA0E045F0D772a851BC7e48357Bcaab0a0795D"),
		Signers: []common.Address{
			common.HexToAddress("0x4769bcaD07e3b938B7f43EB7D278Bc7Cb9efFb38"), // Peter
			common.HexToAddress("0x78d1aD571A1A09D60D9BBf25894b44e4C8859595"), // Martin
			common.HexToAddress("0x286834935f4A8Cfb4FF4C77D5770C2775aE2b0E7"), // Zsolt
			common.HexToAddress("0xb86e2B0Ab5A4B1373e40c51A7C712c70Ba2f9f8E"), // Gary
			common.HexToAddress("0x0DF8fa387C602AE62559cC4aFa4972A7045d6707"), // Guillaume
		},
		Threshold: 2,
	}

	CalaverasChainConfig = &ChainConfig{
		ChainID:             big.NewInt(123),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      true,
		EIP150Block:         big.NewInt(0),
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    nil,
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(500),
		Clique: &CliqueConfig{
			Period: 30,
			Epoch:  30000,
		},
	}

	// VictestChainConfig is the chain parameters to run a Viction Testnet node.
	VictestChainConfig = readChainSpec("chainspecs/victest.json")

	// AllEthashProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Ethash consensus.
	AllEthashProtocolChanges = &ChainConfig{
		ChainID:             big.NewInt(1337),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.Hash{},
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         nil,
		CatalystBlock:       nil,
		Ethash:              new(EthashConfig),
		Clique:              nil,
		Posv:                nil,
	}

	// AllCliqueProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the Clique consensus.
	AllCliqueProtocolChanges = &ChainConfig{
		ChainID:             big.NewInt(1337),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.Hash{},
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         nil,
		LondonBlock:         nil,
		CatalystBlock:       nil,
		Ethash:              nil,
		Clique:              &CliqueConfig{Period: 0, Epoch: 30000},
		Posv:                nil,
	}

	// AllPosvProtocolChanges contains every protocol change (EIPs) introduced
	// and accepted by the Ethereum core developers into the PoSV consensus.
	AllPosvProtocolChanges = &ChainConfig{
		ChainID:               big.NewInt(1337),
		HomesteadBlock:        big.NewInt(0),
		DAOForkBlock:          nil,
		DAOForkSupport:        false,
		EIP150Block:           big.NewInt(0),
		EIP155Block:           big.NewInt(0),
		EIP158Block:           big.NewInt(0),
		ByzantiumBlock:        big.NewInt(0),
		TIP2019Block:          big.NewInt(0),
		TIPSigningBlock:       big.NewInt(0),
		TIPRandomizeBlock:     big.NewInt(0),
		TIPBlacklistBlock:     big.NewInt(0),
		TIPGasPriceBlock:      big.NewInt(0),
		TIPSignerCheckBlock:   big.NewInt(0),
		TIPNativeTradingBlock: big.NewInt(0),
		TIPNativeLendingBlock: big.NewInt(0),
		TIP2021Block:          big.NewInt(0),
		ConstantinopleBlock:   big.NewInt(0),
		PetersburgBlock:       big.NewInt(0),
		IstanbulBlock:         big.NewInt(0),
		MuirGlacierBlock:      big.NewInt(0),
		SaigonBlock:           big.NewInt(0),
		AtlasBlock:            big.NewInt(0),
		PostAtlasBlock:        big.NewInt(0),
		BerlinBlock:           big.NewInt(0),
		LondonBlock:           big.NewInt(0),
		Ethash:                nil,
		Posv:                  &PosvConfig{Period: 2, Epoch: 900, Gap: 5},
	}

	TestChainConfig = &ChainConfig{
		ChainID:             big.NewInt(1),
		HomesteadBlock:      big.NewInt(0),
		DAOForkBlock:        nil,
		DAOForkSupport:      false,
		EIP150Block:         big.NewInt(0),
		EIP150Hash:          common.Hash{},
		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		CatalystBlock:       nil,
		Ethash:              new(EthashConfig),
		Clique:              nil,
		Posv:                nil,
	}
	TestRules = TestChainConfig.Rules(new(big.Int))
)

// TrustedCheckpoint represents a set of post-processed trie roots (CHT and
// BloomTrie) associated with the appropriate section index and head hash. It is
// used to start light syncing from this checkpoint and avoid downloading the
// entire header chain while still being able to securely access old headers/logs.
type TrustedCheckpoint struct {
	SectionIndex uint64      `json:"sectionIndex"`
	SectionHead  common.Hash `json:"sectionHead"`
	CHTRoot      common.Hash `json:"chtRoot"`
	BloomRoot    common.Hash `json:"bloomRoot"`
}

// HashEqual returns an indicator comparing the itself hash with given one.
func (c *TrustedCheckpoint) HashEqual(hash common.Hash) bool {
	if c.Empty() {
		return hash == common.Hash{}
	}
	return c.Hash() == hash
}

// Hash returns the hash of checkpoint's four key fields(index, sectionHead, chtRoot and bloomTrieRoot).
func (c *TrustedCheckpoint) Hash() common.Hash {
	var sectionIndex [8]byte
	binary.BigEndian.PutUint64(sectionIndex[:], c.SectionIndex)

	w := sha3.NewLegacyKeccak256()
	w.Write(sectionIndex[:])
	w.Write(c.SectionHead[:])
	w.Write(c.CHTRoot[:])
	w.Write(c.BloomRoot[:])

	var h common.Hash
	w.Sum(h[:0])
	return h
}

// Empty returns an indicator whether the checkpoint is regarded as empty.
func (c *TrustedCheckpoint) Empty() bool {
	return c.SectionHead == (common.Hash{}) || c.CHTRoot == (common.Hash{}) || c.BloomRoot == (common.Hash{})
}

// CheckpointOracleConfig represents a set of checkpoint contract(which acts as an oracle)
// config which used for light client checkpoint syncing.
type CheckpointOracleConfig struct {
	Address   common.Address   `json:"address"`
	Signers   []common.Address `json:"signers"`
	Threshold uint64           `json:"threshold"`
}

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	ChainID *big.Int `json:"chainId"` // chainId identifies the current chain and is used for replay protection

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"` // Homestead switch block (nil = no fork, 0 = already homestead)

	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`   // TheDAO hard-fork switch block (nil = no fork)
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"` // Whether the nodes supports or opposes the DAO hard-fork

	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int    `json:"eip150Block,omitempty"` // EIP150 HF block (nil = no fork)
	EIP150Hash  common.Hash `json:"eip150Hash,omitempty"`  // EIP150 HF hash (needed for header only clients as only gas pricing changed)

	EIP155Block *big.Int `json:"eip155Block,omitempty"` // EIP155 HF block
	EIP158Block *big.Int `json:"eip158Block,omitempty"` // EIP158 HF block

	ByzantiumBlock      *big.Int `json:"byzantiumBlock,omitempty"`      // Byzantium switch block (nil = no fork, 0 = already on byzantium)
	ConstantinopleBlock *big.Int `json:"constantinopleBlock,omitempty"` // Constantinople switch block (nil = no fork, 0 = already activated)
	PetersburgBlock     *big.Int `json:"petersburgBlock,omitempty"`     // Petersburg switch block (nil = same as Constantinople)
	IstanbulBlock       *big.Int `json:"istanbulBlock,omitempty"`       // Istanbul switch block (nil = no fork, 0 = already on istanbul)
	MuirGlacierBlock    *big.Int `json:"muirGlacierBlock,omitempty"`    // Eip-2384 (bomb delay) switch block (nil = no fork, 0 = already activated)
	BerlinBlock         *big.Int `json:"berlinBlock,omitempty"`         // Berlin switch block (nil = no fork, 0 = already on berlin)
	LondonBlock         *big.Int `json:"londonBlock,omitempty"`         // London switch block (nil = no fork, 0 = already on london)

	CatalystBlock *big.Int `json:"catalystBlock,omitempty"` // Catalyst switch block (nil = no fork, 0 = already on catalyst)

	// Viction network upgrades
	TIP2019Block          *big.Int `json:"tip2019Block,omitempty"`
	TIPSigningBlock       *big.Int `json:"tipSigningBlock,omitempty"`
	TIPRandomizeBlock     *big.Int `json:"tipRandomizeBlock,omitempty"`
	TIPBlacklistBlock     *big.Int `json:"tipBlacklistBlock,omitempty"`
	TIPGasPriceBlock      *big.Int `json:"tipGasPriceBlock,omitempty"`
	TIPSignerCheckBlock   *big.Int `json:"tipSignerCheckBlock,omitempty"`
	TIPNativeTradingBlock *big.Int `json:"tipNativeTradingBlock,omitempty"`
	TIPNativeLendingBlock *big.Int `json:"tipNativeLendingBlock,omitempty"`
	TIP2021Block          *big.Int `json:"tip2021Block,omitempty"`

	SaigonBlock    *big.Int `json:"saigonBlock,omitempty"`
	AtlasBlock     *big.Int `json:"atlasBlock,omitempty"`
	PostAtlasBlock *big.Int `json:"postAtlasBlock,omitempty"`

	// Various consensus engines
	Ethash  *EthashConfig  `json:"ethash,omitempty"`
	Clique  *CliqueConfig  `json:"clique,omitempty"`
	Posv    *PosvConfig    `json:"posv,omitempty"`
	Viction *VictionConfig `json:"viction,omitempty"`
}

// EthashConfig is the consensus engine configs for proof-of-work based sealing.
type EthashConfig struct{}

// String implements the stringer interface, returning the consensus engine details.
func (c *EthashConfig) String() string {
	return "ethash"
}

// CliqueConfig is the consensus engine configs for proof-of-authority based sealing.
type CliqueConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
}

// String implements the stringer interface, returning the consensus engine details.
func (c *CliqueConfig) String() string {
	return "clique"
}

// PosvConfig is the consensus engine configs for proof-of-stake based sealing.
type PosvConfig struct {
	Period uint64 `json:"period"` // Number of seconds between blocks to enforce
	Epoch  uint64 `json:"epoch"`  // Epoch length to reset votes and checkpoint
	Gap    uint64 `json:"gap"`    // Number of blocks to prepare for next epoch
}

// String implements the stringer interface, returning the consensus engine details.
func (c *PosvConfig) String() string {
	return "posv"
}

// Number of blocks producer in a year (365 days) based on configured period.
func (c *PosvConfig) BlocksPerYear() uint64 {
	return 31536000 / c.Period
}

// Check if given block number is checkpoint block.
func (c *PosvConfig) IsCheckpointBlock(number uint64) bool {
	return number%c.Epoch == 0
}

// Check if given block number is gap block.
func (c *PosvConfig) IsGapBlock(number uint64) bool {
	return number%c.Epoch == c.Epoch-c.Gap
}

// String implements the fmt.Stringer interface.
func (c *ChainConfig) String() string {
	var engine interface{}
	switch {
	case c.Ethash != nil:
		engine = c.Ethash
	case c.Clique != nil:
		engine = c.Clique
	case c.Posv != nil:
		engine = c.Posv
	default:
		engine = "unknown"
	}
	return fmt.Sprintf("{ChainID: %v Homestead: %v DAO: %v DAOSupport: %v EIP150: %v EIP155: %v EIP158: %v Byzantium: TIP2019: %v TIPSigning: %v TIPRandomize: %v TIPGasPrice: %v TIPNativeTrading: %v TIPNativeLending: %v TIP2021: %v %v Constantinople: %v Petersburg: %v Istanbul: %v Muir Glacier: %v Saigon: %v Atlas: %v PostAtlas: %v Berlin: %v London: %v Engine: %v}",
		c.ChainID,
		c.HomesteadBlock,
		c.DAOForkBlock,
		c.DAOForkSupport,
		c.EIP150Block,
		c.EIP155Block,
		c.EIP158Block,
		c.ByzantiumBlock,
		c.TIP2019Block,
		c.TIPSigningBlock,
		c.TIPRandomizeBlock,
		c.TIPGasPriceBlock,
		c.TIPNativeTradingBlock,
		c.TIPNativeLendingBlock,
		c.TIP2021Block,
		c.ConstantinopleBlock,
		c.PetersburgBlock,
		c.IstanbulBlock,
		c.MuirGlacierBlock,
		c.SaigonBlock,
		c.AtlasBlock,
		c.PostAtlasBlock,
		c.BerlinBlock,
		c.LondonBlock,
		engine,
	)
}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return isForked(c.HomesteadBlock, num)
}

// IsDAOFork returns whether num is either equal to the DAO fork block or greater.
func (c *ChainConfig) IsDAOFork(num *big.Int) bool {
	return isForked(c.DAOForkBlock, num)
}

// IsEIP150 returns whether num is either equal to the EIP150 fork block or greater.
func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	return isForked(c.EIP150Block, num)
}

// IsEIP155 returns whether num is either equal to the EIP155 fork block or greater.
func (c *ChainConfig) IsEIP155(num *big.Int) bool {
	return isForked(c.EIP155Block, num)
}

// IsEIP158 returns whether num is either equal to the EIP158 fork block or greater.
func (c *ChainConfig) IsEIP158(num *big.Int) bool {
	return isForked(c.EIP158Block, num)
}

// IsByzantium returns whether num is either equal to the Byzantium fork block or greater.
func (c *ChainConfig) IsByzantium(num *big.Int) bool {
	return isForked(c.ByzantiumBlock, num)
}

// IsTIP2019 returns whether num is either equal to the TIP2019 fork block or greater.
func (c *ChainConfig) IsTIP2019(num *big.Int) bool {
	return isForked(c.TIP2019Block, num)
}

// IsTIPSigning returns whether num is either equal to the TIPSigning fork block or greater.
func (c *ChainConfig) IsTIPSigning(num *big.Int) bool {
	return isForked(c.TIPSigningBlock, num)
}

// IsTIPRandomize returns whether num is either equal to the TIPRandomize fork block or greater.
func (c *ChainConfig) IsTIPRandomize(num *big.Int) bool {
	return isForked(c.TIPRandomizeBlock, num)
}

// IsTIPBlacklist returns whether num is either equal to the TIPBlacklist fork block or greater.
func (c *ChainConfig) IsTIPBlacklist(num *big.Int) bool {
	return isForked(c.TIPBlacklistBlock, num)
}

// IsTIPGasPrice returns whether num is either equal to the TIPGasPrice fork block or greater.
func (c *ChainConfig) IsTIPGasPrice(num *big.Int) bool {
	return isForked(c.TIPGasPriceBlock, num)
}

// IsTIPSignerCheck returns whether num is either equal to the TIPSignerCheck fork block or greater.
func (c *ChainConfig) IsTIPSignerCheck(num *big.Int) bool {
	return isForked(c.TIPSignerCheckBlock, num)
}

// IsTIPNativeTrading returns whether num is either equal to the TIPNativeTrading fork block or greater.
func (c *ChainConfig) IsTIPNativeTrading(num *big.Int) bool {
	return isForked(c.TIPNativeTradingBlock, num)
}

// IsTIPNativeLending returns whether num is either equal to the TIPNativeLending fork block or greater.
func (c *ChainConfig) IsTIPNativeLending(num *big.Int) bool {
	return isForked(c.TIPNativeLendingBlock, num)
}

// IsZeroGasEnabled returns true when TomoXTrading or Atlas is active.
func (c *ChainConfig) IsZeroGasEnabled(num *big.Int) bool {
	return isForked(c.TIPNativeTradingBlock, num) || isForked(c.AtlasBlock, num)
}

// IsNativeTradingEnabled returns true when TomoXTrading order processing is active: after TIPTomoX and before Atlas.
func (c *ChainConfig) IsNativeTradingEnabled(num *big.Int) bool {
	return isForked(c.TIPNativeTradingBlock, num) && !isForked(c.AtlasBlock, num)
}

// IsNativeLendingEnabled returns true when TomoXLending order processing is active: after TIPTomoXLending and before Atlas.
func (c *ChainConfig) IsNativeLendingEnabled(num *big.Int) bool {
	return isForked(c.TIPNativeLendingBlock, num) && !isForked(c.AtlasBlock, num)
}

// IsTIP2021 returns whether num is either equal to the TIP2021 fork block or greater.
func (c *ChainConfig) IsTIP2021(num *big.Int) bool {
	return isForked(c.TIP2021Block, num)
}

// IsConstantinople returns whether num is either equal to the Constantinople fork block or greater.
func (c *ChainConfig) IsConstantinople(num *big.Int) bool {
	return isForked(c.ConstantinopleBlock, num)
}

// IsMuirGlacier returns whether num is either equal to the Muir Glacier (EIP-2384) fork block or greater.
func (c *ChainConfig) IsMuirGlacier(num *big.Int) bool {
	return isForked(c.MuirGlacierBlock, num)
}

// IsPetersburg returns whether num is either
// - equal to or greater than the PetersburgBlock fork block,
// - OR is nil, and Constantinople is active
func (c *ChainConfig) IsPetersburg(num *big.Int) bool {
	return isForked(c.PetersburgBlock, num) || c.PetersburgBlock == nil && isForked(c.ConstantinopleBlock, num)
}

// IsIstanbul returns whether num is either equal to the Istanbul fork block or greater.
func (c *ChainConfig) IsIstanbul(num *big.Int) bool {
	return isForked(c.IstanbulBlock, num)
}

// IsSaigon returns whether num is either equal to the Saigon fork block or greater.
func (c *ChainConfig) IsSaigon(num *big.Int) bool {
	return isForked(c.SaigonBlock, num)
}

// IsAtlas returns whether num is either equal to the Atlas fork block or greater.
func (c *ChainConfig) IsAtlas(num *big.Int) bool {
	return isForked(c.AtlasBlock, num)
}

// IsPostAtlas returns whether num is either equal to the PostAtlas fork block or greater.
func (c *ChainConfig) IsPostAtlas(num *big.Int) bool {
	return isForked(c.PostAtlasBlock, num)
}

// IsBerlin returns whether num is either equal to the Berlin fork block or greater.
func (c *ChainConfig) IsBerlin(num *big.Int) bool {
	return isForked(c.BerlinBlock, num)
}

// IsLondon returns whether num is either equal to the London fork block or greater.
func (c *ChainConfig) IsLondon(num *big.Int) bool {
	return isForked(c.LondonBlock, num)
}

// IsCatalyst returns whether num is either equal to the Merge fork block or greater.
func (c *ChainConfig) IsCatalyst(num *big.Int) bool {
	return isForked(c.CatalystBlock, num)
}

// CheckCompatible checks whether scheduled fork transitions have been imported
// with a mismatching chain configuration.
func (c *ChainConfig) CheckCompatible(newcfg *ChainConfig, height uint64) *ConfigCompatError {
	bhead := new(big.Int).SetUint64(height)

	// Iterate checkCompatible to find the lowest conflict.
	var lasterr *ConfigCompatError
	for {
		err := c.checkCompatible(newcfg, bhead)
		if err == nil || (lasterr != nil && err.RewindTo == lasterr.RewindTo) {
			break
		}
		lasterr = err
		bhead.SetUint64(err.RewindTo)
	}
	return lasterr
}

// CheckConfigForkOrder checks that we don't "skip" any forks, geth isn't pluggable enough
// to guarantee that forks can be implemented in a different order than on official networks
func (c *ChainConfig) CheckConfigForkOrder() error {
	type fork struct {
		name     string
		block    *big.Int
		optional bool // if true, the fork may be nil and next fork is still allowed
	}
	var lastFork fork
	for _, cur := range []fork{
		{name: "homesteadBlock", block: c.HomesteadBlock},
		{name: "daoForkBlock", block: c.DAOForkBlock, optional: true},
		{name: "eip150Block", block: c.EIP150Block},
		{name: "eip155Block", block: c.EIP155Block},
		{name: "eip158Block", block: c.EIP158Block},
		{name: "byzantiumBlock", block: c.ByzantiumBlock},
		{name: "tip2019Block", block: c.TIP2019Block},
		{name: "tipSigningBlock", block: c.TIPSigningBlock},
		{name: "tipRandomizeBlock", block: c.TIPRandomizeBlock},
		{name: "tipBlacklistBlock", block: c.TIPBlacklistBlock, optional: true},
		{name: "tipGasPriceBlock", block: c.TIPGasPriceBlock},
		{name: "tipSignerCheckBlock", block: c.TIPSignerCheckBlock, optional: true},
		{name: "tipNativeTradingBlock", block: c.TIPNativeTradingBlock, optional: true},
		{name: "tipNativeLendingBlock", block: c.TIPNativeLendingBlock, optional: true},
		{name: "tip2021Block", block: c.TIP2021Block, optional: true},
		{name: "constantinopleBlock", block: c.ConstantinopleBlock},
		{name: "petersburgBlock", block: c.PetersburgBlock},
		{name: "istanbulBlock", block: c.IstanbulBlock},
		{name: "muirGlacierBlock", block: c.MuirGlacierBlock, optional: true},
		{name: "saigonBlock", block: c.SaigonBlock},
		{name: "atlasBlock", block: c.AtlasBlock},
		{name: "postAtlasBlock", block: c.PostAtlasBlock},
		{name: "berlinBlock", block: c.BerlinBlock},
		{name: "londonBlock", block: c.LondonBlock},
	} {
		// Skip Viction own hard fork for non-Viction networks.
		if c.Posv == nil && isVictionHardfork(cur.name) {
			continue
		}
		if lastFork.name != "" {
			// Next one must be higher number
			if lastFork.block == nil && cur.block != nil {
				return fmt.Errorf("unsupported fork ordering: %v not enabled, but %v enabled at %v",
					lastFork.name, cur.name, cur.block)
			}
			if lastFork.block != nil && cur.block != nil {
				if lastFork.block.Cmp(cur.block) > 0 {
					return fmt.Errorf("unsupported fork ordering: %v enabled at %v, but %v enabled at %v",
						lastFork.name, lastFork.block, cur.name, cur.block)
				}
			}
		}
		// If it was optional and not set, then ignore it
		if !cur.optional || cur.block != nil {
			lastFork = cur
		}
	}
	return nil
}

func (c *ChainConfig) checkCompatible(newcfg *ChainConfig, head *big.Int) *ConfigCompatError {
	if isForkIncompatible(c.HomesteadBlock, newcfg.HomesteadBlock, head) {
		return newCompatError("Homestead fork block", c.HomesteadBlock, newcfg.HomesteadBlock)
	}
	if isForkIncompatible(c.DAOForkBlock, newcfg.DAOForkBlock, head) {
		return newCompatError("DAO fork block", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if c.IsDAOFork(head) && c.DAOForkSupport != newcfg.DAOForkSupport {
		return newCompatError("DAO fork support flag", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if isForkIncompatible(c.EIP150Block, newcfg.EIP150Block, head) {
		return newCompatError("EIP150 fork block", c.EIP150Block, newcfg.EIP150Block)
	}
	if isForkIncompatible(c.EIP155Block, newcfg.EIP155Block, head) {
		return newCompatError("EIP155 fork block", c.EIP155Block, newcfg.EIP155Block)
	}
	if isForkIncompatible(c.EIP158Block, newcfg.EIP158Block, head) {
		return newCompatError("EIP158 fork block", c.EIP158Block, newcfg.EIP158Block)
	}
	if c.IsEIP158(head) && !configNumEqual(c.ChainID, newcfg.ChainID) {
		return newCompatError("EIP158 chain ID", c.EIP158Block, newcfg.EIP158Block)
	}
	if isForkIncompatible(c.ByzantiumBlock, newcfg.ByzantiumBlock, head) {
		return newCompatError("Byzantium fork block", c.ByzantiumBlock, newcfg.ByzantiumBlock)
	}
	if isForkIncompatible(c.TIP2019Block, newcfg.TIP2019Block, head) {
		return newCompatError("TIP2019 fork block", c.TIP2019Block, newcfg.TIP2019Block)
	}
	if isForkIncompatible(c.TIPSigningBlock, newcfg.TIPSigningBlock, head) {
		return newCompatError("TIPSigning fork block", c.TIPSigningBlock, newcfg.TIPSigningBlock)
	}
	if isForkIncompatible(c.TIPRandomizeBlock, newcfg.TIPRandomizeBlock, head) {
		return newCompatError("TIPRandomize fork block", c.TIPRandomizeBlock, newcfg.TIPRandomizeBlock)
	}
	if isForkIncompatible(c.TIPBlacklistBlock, newcfg.TIPBlacklistBlock, head) {
		return newCompatError("TIPBlacklist fork block", c.TIPBlacklistBlock, newcfg.TIPBlacklistBlock)
	}
	if isForkIncompatible(c.TIPGasPriceBlock, newcfg.TIPGasPriceBlock, head) {
		return newCompatError("TIPGasPrice fork block", c.TIPGasPriceBlock, newcfg.TIPGasPriceBlock)
	}
	if isForkIncompatible(c.TIPSignerCheckBlock, newcfg.TIPSignerCheckBlock, head) {
		return newCompatError("TIPSignerCheck fork block", c.TIPSignerCheckBlock, newcfg.TIPSignerCheckBlock)
	}
	if isForkIncompatible(c.TIPNativeTradingBlock, newcfg.TIPNativeTradingBlock, head) {
		return newCompatError("TIPNativeTrading fork block", c.TIPNativeTradingBlock, newcfg.TIPNativeTradingBlock)
	}
	if isForkIncompatible(c.TIPNativeLendingBlock, newcfg.TIPNativeLendingBlock, head) {
		return newCompatError("TIPNativeLending fork block", c.TIPNativeLendingBlock, newcfg.TIPNativeLendingBlock)
	}
	if isForkIncompatible(c.TIP2021Block, newcfg.TIP2021Block, head) {
		return newCompatError("TIP2021 fork block", c.TIP2021Block, newcfg.TIP2021Block)
	}
	if isForkIncompatible(c.ConstantinopleBlock, newcfg.ConstantinopleBlock, head) {
		return newCompatError("Constantinople fork block", c.ConstantinopleBlock, newcfg.ConstantinopleBlock)
	}
	if isForkIncompatible(c.PetersburgBlock, newcfg.PetersburgBlock, head) {
		// the only case where we allow Petersburg to be set in the past is if it is equal to Constantinople
		// mainly to satisfy fork ordering requirements which state that Petersburg fork be set if Constantinople fork is set
		if isForkIncompatible(c.ConstantinopleBlock, newcfg.PetersburgBlock, head) {
			return newCompatError("Petersburg fork block", c.PetersburgBlock, newcfg.PetersburgBlock)
		}
	}
	if isForkIncompatible(c.IstanbulBlock, newcfg.IstanbulBlock, head) {
		return newCompatError("Istanbul fork block", c.IstanbulBlock, newcfg.IstanbulBlock)
	}
	if isForkIncompatible(c.MuirGlacierBlock, newcfg.MuirGlacierBlock, head) {
		return newCompatError("Muir Glacier fork block", c.MuirGlacierBlock, newcfg.MuirGlacierBlock)
	}
	if isForkIncompatible(c.SaigonBlock, newcfg.SaigonBlock, head) {
		return newCompatError("Saigon fork block", c.SaigonBlock, newcfg.SaigonBlock)
	}
	if isForkIncompatible(c.AtlasBlock, newcfg.AtlasBlock, head) {
		return newCompatError("Atlas fork block", c.AtlasBlock, newcfg.AtlasBlock)
	}
	if isForkIncompatible(c.PostAtlasBlock, newcfg.PostAtlasBlock, head) {
		return newCompatError("PostAtlas fork block", c.PostAtlasBlock, newcfg.PostAtlasBlock)
	}
	if isForkIncompatible(c.BerlinBlock, newcfg.BerlinBlock, head) {
		return newCompatError("Berlin fork block", c.BerlinBlock, newcfg.BerlinBlock)
	}
	if isForkIncompatible(c.LondonBlock, newcfg.LondonBlock, head) {
		return newCompatError("London fork block", c.LondonBlock, newcfg.LondonBlock)
	}
	return nil
}

// isForkIncompatible returns true if a fork scheduled at s1 cannot be rescheduled to
// block s2 because head is already past the fork.
func isForkIncompatible(s1, s2, head *big.Int) bool {
	return (isForked(s1, head) || isForked(s2, head)) && !configNumEqual(s1, s2)
}

// isForked returns whether a fork scheduled at block s is active at the given head block.
func isForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

func configNumEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

// ConfigCompatError is raised if the locally-stored blockchain is initialised with a
// ChainConfig that would alter the past.
type ConfigCompatError struct {
	What string
	// block numbers of the stored and new configurations
	StoredConfig, NewConfig *big.Int
	// the block number to which the local chain must be rewound to correct the error
	RewindTo uint64
}

func newCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && rew.Sign() > 0 {
		err.RewindTo = rew.Uint64() - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %s in database (have %d, want %d, rewindto %d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

// Rules wraps ChainConfig and is merely syntactic sugar or can be used for functions
// that do not have or require information about the block.
//
// Rules is a one time interface meaning that it shouldn't be used in between transition
// phases.
type Rules struct {
	ChainID                                                 *big.Int
	IsHomestead, IsEIP150, IsEIP155, IsEIP158               bool
	IsByzantium, IsConstantinople, IsPetersburg, IsIstanbul bool
	IsBerlin, IsLondon, IsCatalyst                          bool

	IsTIP2019, IsTIPSigning, IsTIPRandomize           bool
	IsTIPBlacklist, IsTIPGasPrice, IsTIPSignerCheck   bool
	IsTIPNativeTrading, IsTIPNativeLending, IsTIP2021 bool

	IsSaigon, IsAtlas, IsPostAtlas bool
}

// Rules ensures c's ChainID is not nil.
func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainID := c.ChainID
	if chainID == nil {
		chainID = new(big.Int)
	}
	return Rules{
		ChainID:          new(big.Int).Set(chainID),
		IsHomestead:      c.IsHomestead(num),
		IsEIP150:         c.IsEIP150(num),
		IsEIP155:         c.IsEIP155(num),
		IsEIP158:         c.IsEIP158(num),
		IsByzantium:      c.IsByzantium(num),
		IsConstantinople: c.IsConstantinople(num),
		IsPetersburg:     c.IsPetersburg(num),
		IsIstanbul:       c.IsIstanbul(num),
		IsBerlin:         c.IsBerlin(num),
		IsLondon:         c.IsLondon(num),
		IsCatalyst:       c.IsCatalyst(num),

		IsTIP2019:          c.IsTIP2019(num),
		IsTIPSigning:       c.IsTIPSigning(num),
		IsTIPRandomize:     c.IsTIPRandomize(num),
		IsTIPBlacklist:     c.IsTIPBlacklist(num),
		IsTIPGasPrice:      c.IsTIPGasPrice(num),
		IsTIPSignerCheck:   c.IsTIPSignerCheck(num),
		IsTIPNativeTrading: c.IsTIPNativeTrading(num),
		IsTIPNativeLending: c.IsTIPNativeLending(num),
		IsTIP2021:          c.IsTIP2021(num),

		IsSaigon:    c.IsSaigon(num),
		IsAtlas:     c.IsAtlas(num),
		IsPostAtlas: c.IsPostAtlas(num),
	}
}
