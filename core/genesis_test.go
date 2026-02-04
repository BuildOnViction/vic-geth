// Copyright 2017 The go-ethereum Authors
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

package core

import (
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

func TestDefaultGenesisBlock(t *testing.T) {
	block := DefaultGenesisBlock().ToBlock(nil)
	if block.Hash() != params.MainnetGenesisHash {
		t.Errorf("wrong mainnet genesis hash, got %v, want %v", block.Hash(), params.MainnetGenesisHash)
	}
	block = DefaultRopstenGenesisBlock().ToBlock(nil)
	if block.Hash() != params.RopstenGenesisHash {
		t.Errorf("wrong ropsten genesis hash, got %v, want %v", block.Hash(), params.RopstenGenesisHash)
	}
}

func TestSetupGenesis(t *testing.T) {
	var (
		customghash = common.HexToHash("0x89c99d90b79719238d2645c7642f2c9295246e80775b38cfd162b696817fbd50")
		customg     = Genesis{
			Config: &params.ChainConfig{HomesteadBlock: big.NewInt(3)},
			Alloc: GenesisAlloc{
				{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
			},
		}
		oldcustomg = customg
	)
	oldcustomg.Config = &params.ChainConfig{HomesteadBlock: big.NewInt(2)}
	tests := []struct {
		name       string
		fn         func(ethdb.Database) (*params.ChainConfig, common.Hash, error)
		wantConfig *params.ChainConfig
		wantHash   common.Hash
		wantErr    error
	}{
		{
			name: "genesis without ChainConfig",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				return SetupGenesisBlock(db, new(Genesis))
			},
			wantErr:    errGenesisNoConfig,
			wantConfig: params.AllEthashProtocolChanges,
		},
		{
			name: "no block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   params.MainnetGenesisHash,
			wantConfig: params.MainnetChainConfig,
		},
		{
			name: "mainnet block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				DefaultGenesisBlock().MustCommit(db)
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   params.MainnetGenesisHash,
			wantConfig: params.MainnetChainConfig,
		},
		{
			name: "custom block in DB, genesis == nil",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				customg.MustCommit(db)
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
		},
		{
			name: "custom block in DB, genesis == ropsten",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				customg.MustCommit(db)
				return SetupGenesisBlock(db, DefaultRopstenGenesisBlock())
			},
			wantErr:    &GenesisMismatchError{Stored: customghash, New: params.RopstenGenesisHash},
			wantHash:   params.RopstenGenesisHash,
			wantConfig: params.RopstenChainConfig,
		},
		{
			name: "compatible config in DB",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				oldcustomg.MustCommit(db)
				return SetupGenesisBlock(db, &customg)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
		},
		{
			name: "incompatible config in DB",
			fn: func(db ethdb.Database) (*params.ChainConfig, common.Hash, error) {
				// Commit the 'old' genesis block with Homestead transition at #2.
				// Advance to block #4, past the homestead transition block of customg.
				genesis := oldcustomg.MustCommit(db)

				bc, _ := NewBlockChain(db, nil, oldcustomg.Config, ethash.NewFullFaker(), vm.Config{}, nil, nil)
				defer bc.Stop()

				blocks, _ := GenerateChain(oldcustomg.Config, genesis, ethash.NewFaker(), db, 4, nil)
				bc.InsertChain(blocks)
				bc.CurrentBlock()
				// This should return a compatibility error.
				return SetupGenesisBlock(db, &customg)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
			wantErr: &params.ConfigCompatError{
				What:         "Homestead fork block",
				StoredConfig: big.NewInt(2),
				NewConfig:    big.NewInt(3),
				RewindTo:     1,
			},
		},
	}

	for _, test := range tests {
		db := rawdb.NewMemoryDatabase()
		config, hash, err := test.fn(db)
		// Check the return values.
		if !reflect.DeepEqual(err, test.wantErr) {
			spew := spew.ConfigState{DisablePointerAddresses: true, DisableCapacities: true}
			t.Errorf("%s: returned error %#v, want %#v", test.name, spew.NewFormatter(err), spew.NewFormatter(test.wantErr))
		}
		if !reflect.DeepEqual(config, test.wantConfig) {
			t.Errorf("%s:\nreturned %v\nwant     %v", test.name, config, test.wantConfig)
		}
		if hash != test.wantHash {
			t.Errorf("%s: returned hash %s, want %s", test.name, hash.Hex(), test.wantHash.Hex())
		} else if err == nil {
			// Check database content.
			stored := rawdb.ReadBlock(db, test.wantHash, 0)
			if stored.Hash() != test.wantHash {
				t.Errorf("%s: block in DB has hash %s, want %s", test.name, stored.Hash(), test.wantHash)
			}
		}
	}
}

// TestGenesisJSONFile tests that the genesis.json file produces the correct genesis block hash
func TestGenesisJSONFile(t *testing.T) {
	// Find the genesis.json file relative to the test file
	// The test file is in core/, so we need to go up one level
	testDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Try to find genesis.json in the project root
	// We're in core/, so go up one level
	projectRoot := filepath.Dir(testDir)
	genesisPath := filepath.Join(projectRoot, "genesis.json")

	// If not found, try going up one more level (in case we're in a subdirectory)
	if _, err := os.Stat(genesisPath); os.IsNotExist(err) {
		projectRoot = filepath.Dir(projectRoot)
		genesisPath = filepath.Join(projectRoot, "genesis.json")
	}

	// Read the genesis.json file
	data, err := ioutil.ReadFile(genesisPath)
	if err != nil {
		t.Skipf("genesis.json file not found at %s, skipping test: %v", genesisPath, err)
		return
	}

	// Unmarshal the genesis JSON
	var genesis Genesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		t.Fatalf("Failed to unmarshal genesis.json: %v", err)
	}

	// Convert to block
	block := genesis.ToBlock(nil)

	// Check that the hash matches the expected Viction mainnet hash
	expectedHash := params.VicMainnetGenesisHash
	actualHash := block.Hash()

	if actualHash != expectedHash {
		t.Errorf("Wrong genesis hash from genesis.json:\n  got:  %s\n  want: %s",
			actualHash.Hex(), expectedHash.Hex())

		// Also log additional debug information
		t.Logf("Block details:")
		t.Logf("  Number: %d", block.Number())
		t.Logf("  StateRoot: %s", block.Root().Hex())
		t.Logf("  Timestamp: %d", block.Time())
		t.Logf("  GasLimit: %d", block.GasLimit())
		t.Logf("  Difficulty: %s", block.Difficulty().String())
		t.Logf("  ExtraData length: %d", len(block.Extra()))
		t.Logf("  Alloc accounts: %d", len(genesis.Alloc))
	} else {
		t.Logf("Genesis block hash matches expected Viction mainnet hash: %s", actualHash.Hex())
	}

	// Also verify the state root if we have an expected value
	// The state root should be: 0x1394d6e0a3d48b3d25da2206de068a1444108280c60d360bd9d5a870004529ee
	expectedStateRoot := common.HexToHash("0x1394d6e0a3d48b3d25da2206de068a1444108280c60d360bd9d5a870004529ee")
	actualStateRoot := block.Root()

	if actualStateRoot != expectedStateRoot {
		t.Errorf("Wrong state root from genesis.json:\n  got:  %s\n  want: %s",
			actualStateRoot.Hex(), expectedStateRoot.Hex())
	} else {
		t.Logf("State root matches expected: %s", actualStateRoot.Hex())
	}
}

// TestVictionMainnetGenesisBlock tests the default Viction mainnet genesis block
func TestVictionMainnetGenesisBlock(t *testing.T) {
	block := DefaultVicMainnetGenesisBlock().ToBlock(nil)
	if block.Hash() != params.VicMainnetGenesisHash {
		t.Errorf("wrong Viction mainnet genesis hash, got %v, want %v",
			block.Hash().String(), params.VicMainnetGenesisHash.String())
	} else {
		t.Logf("Viction mainnet genesis block hash matches: %s", block.Hash().Hex())
	}

	// Test that the expected hash constant is correct
	expectedHash := common.HexToHash("0x9326145f8a2c8c00bbe13afc7d7f3d9c868b5ef39d89f2f4e9390e9720298624")
	if params.VicMainnetGenesisHash != expectedHash {
		t.Errorf("params.VicMainnetGenesisHash constant is wrong: got %s, want %s",
			params.VicMainnetGenesisHash.Hex(), expectedHash.Hex())
	}
}
