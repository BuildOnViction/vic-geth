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

package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

const (
	finalityTestValidatorCount = 8  // validators in the genesis checkpoint; the next key is an outsider
	finalityTestOutsider       = 8  // index of the key outside every validator set
	finalityTestWindow         = 30 // ConsensusLimitTimeFinality of the test chains
)

var finalityTestSignContract = common.HexToAddress("0x0000000000000000000000000000000000000089")

func TestPosvFinalityFinalizesAtQuorum(t *testing.T) {
	env := newFinalityTestEnv(t, 900)
	b1 := env.newBlock(env.genesis, finalityTestBlock{creator: 0, attestor: 1})
	b2 := env.newBlock(b1, finalityTestBlock{creator: 2, attestor: 3, votes: votesFor(b1, 2, 3, 4)})
	b3 := env.newBlock(b2, finalityTestBlock{creator: 4, attestor: 5, votes: votesFor(b1, 5)})

	env.extend(b1, b2)
	belowQuorum := env.tracker.finalizedHeader()
	env.extend(b3)

	if belowQuorum != nil {
		t.Fatalf("5 of 8 signers finalized block %d", belowQuorum.Number)
	}
	env.requireFinalized(b1)
	if stored := rawdb.ReadPosvFinalizedBlockHash(env.db); stored != b1.Hash() {
		t.Fatalf("stored finalized hash mismatch: have %x, want %x", stored, b1.Hash())
	}
}

func TestPosvFinalityIgnoresInvalidVote(t *testing.T) {
	tests := []struct {
		name          string
		vote          func(b1 *types.Block) finalityTestVote
		wantFinalized bool
	}{
		{
			name:          "sixth validator finalizes",
			vote:          func(b1 *types.Block) finalityTestVote { return finalityTestVote{signer: 5, target: b1} },
			wantFinalized: true,
		},
		{
			name: "failed BlockSigner transaction is not counted",
			vote: func(b1 *types.Block) finalityTestVote { return finalityTestVote{signer: 5, target: b1, failed: true} },
		},
		{
			name: "sender outside the validator set is not counted",
			vote: func(b1 *types.Block) finalityTestVote {
				return finalityTestVote{signer: finalityTestOutsider, target: b1}
			},
		},
		{
			name: "repeated vote of a validator counts once",
			vote: func(b1 *types.Block) finalityTestVote { return finalityTestVote{signer: 4, target: b1} },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newFinalityTestEnv(t, 900)
			b1 := env.newBlock(env.genesis, finalityTestBlock{creator: 0, attestor: 1})
			votes := append(votesFor(b1, 2, 3, 4), tt.vote(b1))
			b2 := env.newBlock(b1, finalityTestBlock{creator: 2, attestor: 3, votes: votes})

			env.extend(b1, b2)

			if tt.wantFinalized {
				env.requireFinalized(b1)
			} else {
				env.requireNotFinalized()
			}
		})
	}
}

func TestPosvFinalityIgnoresVotesForNonCanonicalBlock(t *testing.T) {
	env := newFinalityTestEnv(t, 900)
	b1 := env.newBlock(env.genesis, finalityTestBlock{creator: 0, attestor: 1})
	side := env.newBlock(env.genesis, finalityTestBlock{creator: 2, attestor: 3})
	b2 := env.newBlock(b1, finalityTestBlock{creator: 2, attestor: 3, votes: votesFor(side, 0, 1, 2, 3, 4, 5, 6, 7)})

	env.extend(b1, b2)

	env.requireNotFinalized()
}

func TestPosvFinalityWindow(t *testing.T) {
	tests := []struct {
		name          string
		includedAt    uint64
		wantFinalized bool
	}{
		{name: "vote at the last block of the window counts", includedAt: 1 + finalityTestWindow, wantFinalized: true},
		{name: "vote after the window is not counted", includedAt: 2 + finalityTestWindow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newFinalityTestEnv(t, 900)
			b1 := env.newBlock(env.genesis, finalityTestBlock{creator: 0, attestor: 1})
			head := env.newBlock(b1, finalityTestBlock{creator: 2, attestor: 3, votes: votesFor(b1, 2, 3, 4)})
			env.extend(b1, head)
			for head.NumberU64()+1 < tt.includedAt {
				head = env.newBlock(head, finalityTestBlock{creator: 2, attestor: 3})
				env.extend(head)
			}
			last := env.newBlock(head, finalityTestBlock{creator: 2, attestor: 3, votes: votesFor(b1, 5)})

			env.extend(last)

			if tt.wantFinalized {
				env.requireFinalized(b1)
			} else {
				env.requireNotFinalized()
			}
		})
	}
}

func TestPosvFinalityRetalliesAfterReorg(t *testing.T) {
	env := newFinalityTestEnv(t, 900)
	b1 := env.newBlock(env.genesis, finalityTestBlock{creator: 0, attestor: 1})
	droppedVotes := env.newBlock(b1, finalityTestBlock{creator: 2, attestor: 3, votes: votesFor(b1, 2, 3, 4)})
	b2 := env.newBlock(b1, finalityTestBlock{creator: 4, attestor: 5})
	b3 := env.newBlock(b2, finalityTestBlock{creator: 6, attestor: 7, votes: votesFor(b1, 5)})
	b4 := env.newBlock(b3, finalityTestBlock{creator: 0, attestor: 1, votes: votesFor(b1, 2, 3, 4)})
	env.extend(b1, droppedVotes)

	env.extend(b2, b3)
	afterReorg := env.tracker.finalizedHeader()
	env.extend(b4)

	if afterReorg != nil {
		t.Fatalf("votes of a reorganised block finalized block %d", afterReorg.Number)
	}
	env.requireFinalized(b1)
}

func TestPosvFinalityDropsFinalizedBlockReorganisedOut(t *testing.T) {
	env := newFinalityTestEnv(t, 900)
	b1 := env.newBlock(env.genesis, finalityTestBlock{creator: 0, attestor: 1})
	b2 := env.newBlock(b1, finalityTestBlock{creator: 2, attestor: 3, votes: votesFor(b1, 2, 3, 4, 5)})
	fork1 := env.newBlock(env.genesis, finalityTestBlock{creator: 4, attestor: 5})
	env.extend(b1, b2)

	env.extend(fork1)

	env.requireNotFinalized()
	if stored := rawdb.ReadPosvFinalizedBlockHash(env.db); stored != (common.Hash{}) {
		t.Fatalf("dropped finalized block still stored: %x", stored)
	}
}

func TestPosvFinalityLoad(t *testing.T) {
	tests := []struct {
		name          string
		head          func(env *finalityTestEnv, b1, b2 *types.Block) *types.Header
		wantFinalized bool
	}{
		{
			name:          "canonical finalized block is restored",
			head:          func(env *finalityTestEnv, b1, b2 *types.Block) *types.Header { return b2.Header() },
			wantFinalized: true,
		},
		{
			name: "finalized block above the head is dropped",
			head: func(env *finalityTestEnv, b1, b2 *types.Block) *types.Header {
				delete(env.chain.canonical, 2)
				delete(env.chain.canonical, 1)
				return env.genesis.Header()
			},
		},
		{
			name: "finalized block off the canonical chain is dropped",
			head: func(env *finalityTestEnv, b1, b2 *types.Block) *types.Header {
				fork1 := env.newBlock(env.genesis, finalityTestBlock{creator: 4, attestor: 5})
				env.chain.canonical[1] = fork1.Hash()
				delete(env.chain.canonical, 2)
				return fork1.Header()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newFinalityTestEnv(t, 900)
			b1 := env.newBlock(env.genesis, finalityTestBlock{creator: 0, attestor: 1})
			b2 := env.newBlock(b1, finalityTestBlock{creator: 2, attestor: 3, votes: votesFor(b1, 2, 3, 4, 5)})
			env.extend(b1, b2)
			restarted := newPosvFinality(env.chain, env.db, env.engine)
			head := tt.head(env, b1, b2)

			restarted.load(head)

			finalized := restarted.finalizedHeader()
			stored := rawdb.ReadPosvFinalizedBlockHash(env.db)
			if !tt.wantFinalized {
				if finalized != nil || stored != (common.Hash{}) {
					t.Fatalf("finalized block kept: loaded %v, stored %x", finalized, stored)
				}
				return
			}
			if finalized == nil || finalized.Hash() != b1.Hash() || stored != b1.Hash() {
				t.Fatalf("finalized block not restored: loaded %v, stored %x, want %x", finalized, stored, b1.Hash())
			}
		})
	}
}

func TestPosvFinalityUsesEpochCheckpointValidators(t *testing.T) {
	const epoch = 10
	env := newFinalityTestEnv(t, epoch)
	head := env.genesis
	for head.NumberU64()+1 < epoch {
		head = env.newBlock(head, finalityTestBlock{creator: 0, attestor: 1})
		env.extend(head)
	}
	checkpoint := env.newBlock(head, finalityTestBlock{creator: 4, attestor: 5, checkpointValidators: []int{4, 5, 6, 7}})
	signed := env.newBlock(checkpoint, finalityTestBlock{creator: 4, attestor: 5})
	oldSetVote := env.newBlock(signed, finalityTestBlock{creator: 6, attestor: 7, votes: votesFor(signed, 0)})
	newSetVote := env.newBlock(oldSetVote, finalityTestBlock{creator: 4, attestor: 5, votes: votesFor(signed, 6)})
	env.extend(checkpoint, signed)

	env.extend(oldSetVote)
	afterOldSetVote := env.tracker.finalizedHeader()
	env.extend(newSetVote)

	if afterOldSetVote != nil {
		t.Fatalf("vote of a validator outside the epoch finalized block %d", afterOldSetVote.Number)
	}
	env.requireFinalized(signed)
}

func TestNewPosvFinalityRequiresPosvChain(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	tests := []struct {
		name    string
		mutate  func(config *params.ChainConfig)
		wantNil bool
	}{
		{name: "PoSV chain with a finality window is tracked"},
		{name: "chain without PoSV config", mutate: func(config *params.ChainConfig) { config.Posv = nil }, wantNil: true},
		{name: "chain without Viction config", mutate: func(config *params.ChainConfig) { config.Viction = nil }, wantNil: true},
		{
			name:    "chain without finality window",
			mutate:  func(config *params.ChainConfig) { config.Viction.ConsensusLimitTimeFinality = 0 },
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := newFinalityTestConfig(params.TestChainConfig, 900)
			if tt.mutate != nil {
				tt.mutate(config)
			}

			tracker := newPosvFinality(&finalityTestChain{config: config}, db, posv.New(&params.PosvConfig{Epoch: 900}, db))

			if (tracker == nil) != tt.wantNil {
				t.Fatalf("tracker nil: have %v, want %v", tracker == nil, tt.wantNil)
			}
		})
	}
	t.Run("non-PoSV engine", func(t *testing.T) {
		config := newFinalityTestConfig(params.TestChainConfig, 900)

		tracker := newPosvFinality(&finalityTestChain{config: config}, db, ethash.NewFaker())

		if tracker != nil {
			t.Fatal("tracker created for a non-PoSV engine")
		}
	})
}

// Insert blocks into a real chain and check that the canonical-head and state-reload hooks drive the tracker.
func TestPosvFinalityTracksBlockChain(t *testing.T) {
	keys := makeFinalityTestKeys(4)
	addrs := finalityTestAddresses(keys)
	alloc := GenesisAlloc{}
	for _, addr := range addrs {
		alloc[addr] = GenesisAccount{Balance: big.NewInt(params.Ether)}
	}
	var (
		db      = rawdb.NewMemoryDatabase()
		engine  = ethash.NewFaker()
		gspec   = &Genesis{Config: params.TestChainConfig, ExtraData: makeCheckpointExtra(addrs), Alloc: alloc}
		genesis = gspec.MustCommit(db)
		signer  = types.LatestSigner(params.TestChainConfig)
	)
	blocks, _ := GenerateChain(params.TestChainConfig, genesis, engine, db, 2, func(i int, b *BlockGen) {
		if i != 1 {
			return
		}
		target := b.PrevBlock(0)
		for _, key := range keys[:3] {
			addr := crypto.PubkeyToAddress(key.PublicKey)
			tx := types.NewTransaction(b.TxNonce(addr), finalityTestSignContract, common.Big0, 100000, b.BaseFee(), makeFinalityTestSignData(target))
			signed, err := types.SignTx(tx, signer, key)
			if err != nil {
				t.Fatalf("failed to sign vote: %v", err)
			}
			b.AddTx(signed)
		}
	})
	bc, err := NewBlockChain(db, nil, params.TestChainConfig, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	defer bc.Stop()
	posvChain := finalityTestConfigChain{BlockChain: bc, config: newFinalityTestConfig(params.TestChainConfig, 900)}
	bc.posvFinality = newPosvFinality(posvChain, db, posv.New(posvChain.config.Posv, db))
	bc.posvFinality.load(bc.CurrentBlock().Header())

	if _, err := bc.InsertChain(blocks); err != nil {
		t.Fatalf("failed to insert blocks: %v", err)
	}
	inserted := bc.posvFinality.finalizedHeader()
	if err := bc.SetHead(0); err != nil {
		t.Fatalf("failed to rewind: %v", err)
	}

	if inserted == nil || inserted.Hash() != blocks[0].Hash() {
		t.Fatalf("inserted votes did not finalize block 1: have %v", inserted)
	}
	if rewound := bc.posvFinality.finalizedHeader(); rewound != nil {
		t.Fatalf("rewind below the finalized block kept it: %d", rewound.Number)
	}
	if stored := rawdb.ReadPosvFinalizedBlockHash(db); stored != (common.Hash{}) {
		t.Fatalf("rewind below the finalized block kept it stored: %x", stored)
	}
}

// finalityTestChain is an in-memory chain whose canonical mapping the tests set by hand.
type finalityTestChain struct {
	config    *params.ChainConfig
	headers   map[common.Hash]*types.Header
	blocks    map[common.Hash]*types.Block
	receipts  map[common.Hash]types.Receipts
	canonical map[uint64]common.Hash
}

func (c *finalityTestChain) Config() *params.ChainConfig { return c.config }

func (c *finalityTestChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	if block, ok := c.blocks[hash]; ok && block.NumberU64() == number {
		return block
	}
	return nil
}

func (c *finalityTestChain) GetCanonicalHash(number uint64) common.Hash { return c.canonical[number] }

func (c *finalityTestChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	if header, ok := c.headers[hash]; ok && header.Number.Uint64() == number {
		return header
	}
	return nil
}

func (c *finalityTestChain) GetHeaderByHash(hash common.Hash) *types.Header { return c.headers[hash] }

func (c *finalityTestChain) GetReceiptsByHash(hash common.Hash) types.Receipts {
	return c.receipts[hash]
}

// finalityTestConfigChain presents a BlockChain to the tracker under a PoSV chain config, so ethash test chains can carry votes.
type finalityTestConfigChain struct {
	*BlockChain
	config *params.ChainConfig
}

func (c finalityTestConfigChain) Config() *params.ChainConfig { return c.config }

// finalityTestVote is a BlockSigner transaction of the validator key at index signer for target.
type finalityTestVote struct {
	signer int
	target *types.Block
	failed bool
}

// finalityTestBlock describes a block by the key indexes of its creator and attestor.
type finalityTestBlock struct {
	creator              int
	attestor             int
	checkpointValidators []int // validator key indexes written into a checkpoint block's extra-data
	votes                []finalityTestVote
}

type finalityTestEnv struct {
	t       *testing.T
	keys    []*ecdsa.PrivateKey
	chain   *finalityTestChain
	db      ethdb.Database
	engine  *posv.Posv
	tracker *posvFinality
	genesis *types.Block
}

func newFinalityTestEnv(t *testing.T, epoch uint64) *finalityTestEnv {
	t.Helper()
	keys := makeFinalityTestKeys(finalityTestValidatorCount + 1)
	config := newFinalityTestConfig(params.AllPosvProtocolChanges, epoch)
	genesis := types.NewBlockWithHeader(&types.Header{
		Number:     big.NewInt(0),
		Difficulty: big.NewInt(1),
		Extra:      makeCheckpointExtra(finalityTestAddresses(keys[:finalityTestValidatorCount])),
	})
	chain := &finalityTestChain{
		config:    config,
		headers:   map[common.Hash]*types.Header{genesis.Hash(): genesis.Header()},
		blocks:    map[common.Hash]*types.Block{genesis.Hash(): genesis},
		receipts:  map[common.Hash]types.Receipts{},
		canonical: map[uint64]common.Hash{0: genesis.Hash()},
	}
	db := rawdb.NewMemoryDatabase()
	engine := posv.New(config.Posv, db)
	tracker := newPosvFinality(chain, db, engine)
	tracker.load(genesis.Header())
	return &finalityTestEnv{t: t, keys: keys, chain: chain, db: db, engine: engine, tracker: tracker, genesis: genesis}
}

// Build a sealed and attested child of parent carrying the given votes, without making it canonical.
func (env *finalityTestEnv) newBlock(parent *types.Block, spec finalityTestBlock) *types.Block {
	env.t.Helper()
	header := &types.Header{
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Difficulty: big.NewInt(int64(spec.creator + 1)),
		Time:       parent.Time() + 2,
		Extra:      make([]byte, posv.ExtraVanity+posv.ExtraSeal),
		Posv:       true,
	}
	if spec.checkpointValidators != nil {
		validators := make([]common.Address, len(spec.checkpointValidators))
		for i, index := range spec.checkpointValidators {
			validators[i] = crypto.PubkeyToAddress(env.keys[index].PublicKey)
		}
		header.Extra = makeCheckpointExtra(validators)
	}
	signer := types.MakeSigner(env.chain.config, header.Number)
	txs := make([]*types.Transaction, len(spec.votes))
	receipts := make(types.Receipts, len(spec.votes))
	for i, vote := range spec.votes {
		tx := types.NewTransaction(uint64(i), finalityTestSignContract, common.Big0, 200000, common.Big0, makeFinalityTestSignData(vote.target))
		signed, err := types.SignTx(tx, signer, env.keys[vote.signer])
		if err != nil {
			env.t.Fatalf("failed to sign vote: %v", err)
		}
		txs[i] = signed
		receipts[i] = &types.Receipt{Status: types.ReceiptStatusSuccessful}
		if vote.failed {
			receipts[i].Status = types.ReceiptStatusFailed
		}
	}
	env.seal(header, spec.creator, spec.attestor)
	block := types.NewBlockWithHeader(header).WithBody(txs, nil)
	env.chain.headers[block.Hash()] = block.Header()
	env.chain.blocks[block.Hash()] = block
	env.chain.receipts[block.Hash()] = receipts
	return block
}

// Sign header as its creator and attestor, as PoSV seals and attests blocks.
func (env *finalityTestEnv) seal(header *types.Header, creator, attestor int) {
	env.t.Helper()
	sealHash := posv.SealHash(header).Bytes()
	creatorSig, err := crypto.Sign(sealHash, env.keys[creator])
	if err != nil {
		env.t.Fatalf("failed to seal block: %v", err)
	}
	copy(header.Extra[len(header.Extra)-posv.ExtraSeal:], creatorSig)
	if header.Attestor, err = crypto.Sign(sealHash, env.keys[attestor]); err != nil {
		env.t.Fatalf("failed to attest block: %v", err)
	}
}

// Make each block the canonical head in turn, as the chain does on import and reorg.
func (env *finalityTestEnv) extend(blocks ...*types.Block) {
	for _, block := range blocks {
		env.chain.canonical[block.NumberU64()] = block.Hash()
		env.tracker.applyHead(block)
	}
}

func (env *finalityTestEnv) requireFinalized(want *types.Block) {
	env.t.Helper()
	if have := env.tracker.finalizedHeader(); have == nil || have.Hash() != want.Hash() {
		env.t.Fatalf("finalized block mismatch: have %v, want #%d %x", have, want.NumberU64(), want.Hash())
	}
}

func (env *finalityTestEnv) requireNotFinalized() {
	env.t.Helper()
	if have := env.tracker.finalizedHeader(); have != nil {
		env.t.Fatalf("unexpected finalized block #%d %x", have.Number, have.Hash())
	}
}

func votesFor(target *types.Block, signers ...int) []finalityTestVote {
	votes := make([]finalityTestVote, len(signers))
	for i, signer := range signers {
		votes[i] = finalityTestVote{signer: signer, target: target}
	}
	return votes
}

// Return a copy of base with the PoSV and Viction settings the finality tracker reads.
func newFinalityTestConfig(base *params.ChainConfig, epoch uint64) *params.ChainConfig {
	config := *base
	config.Posv = &params.PosvConfig{Period: 2, Epoch: epoch, Gap: 5}
	config.Viction = &params.VictionConfig{
		ValidatorBlockSignContract: finalityTestSignContract,
		ConsensusLimitTimeFinality: finalityTestWindow,
	}
	return &config
}

func makeFinalityTestKeys(count int) []*ecdsa.PrivateKey {
	keys := make([]*ecdsa.PrivateKey, count)
	for i := range keys {
		keys[i], _ = crypto.ToECDSA(common.LeftPadBytes([]byte{byte(i + 1)}, 32))
	}
	return keys
}

func finalityTestAddresses(keys []*ecdsa.PrivateKey) []common.Address {
	addrs := make([]common.Address, len(keys))
	for i, key := range keys {
		addrs[i] = crypto.PubkeyToAddress(key.PublicKey)
	}
	return addrs
}

// Build checkpoint extra-data: vanity, validator addresses, empty seal.
func makeCheckpointExtra(validators []common.Address) []byte {
	extra := make([]byte, posv.ExtraVanity)
	for _, validator := range validators {
		extra = append(extra, validator.Bytes()...)
	}
	return append(extra, make([]byte, posv.ExtraSeal)...)
}

// Build the calldata of sign(uint256,bytes32) for target.
func makeFinalityTestSignData(target *types.Block) []byte {
	data := common.Hex2Bytes("e341eaa4")
	data = append(data, common.LeftPadBytes(target.Number().Bytes(), 32)...)
	return append(data, target.Hash().Bytes()...)
}
