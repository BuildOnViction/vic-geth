// Copyright 2014 The go-ethereum Authors
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

package types

import (
	"bytes"
	"hash"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

// from bcValidBlockTest.json, "SimpleTx"
func TestBlockEncoding(t *testing.T) {
	blockEnc := common.FromHex("f90260f901f9a083cafc574e1f51ba9dc0568fc617a08ea2429fb384059c972f13b19fa1c8dd55a01dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347948888f1f195afa192cfee860698584c030f4c9db1a0ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017a05fe50b260da6308036625b850b5d6ced6d0a9f814c0688bc91ffb7b7a3a54b67a0bc37d79753ad738a6dac4921e57392f145d8887476de3f783dfa7edae9283e52b90100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000008302000001832fefd8825208845506eb0780a0bd4472abb6659ebe3ee06ee4d7b72a00a9f4d001caca51342001075469aff49888a13a5a8c8f2bb1c4f861f85f800a82c35094095e7baea6a6c7c4c2dfeb977efac326af552d870a801ba09bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094fa08a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b1c0")
	var block Block
	if err := rlp.DecodeBytes(blockEnc, &block); err != nil {
		t.Fatal("decode error: ", err)
	}

	check := func(f string, got, want interface{}) {
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s mismatch: got %v, want %v", f, got, want)
		}
	}
	check("Difficulty", block.Difficulty(), big.NewInt(131072))
	check("GasLimit", block.GasLimit(), uint64(3141592))
	check("GasUsed", block.GasUsed(), uint64(21000))
	check("Coinbase", block.Coinbase(), common.HexToAddress("8888f1f195afa192cfee860698584c030f4c9db1"))
	check("MixDigest", block.MixDigest(), common.HexToHash("bd4472abb6659ebe3ee06ee4d7b72a00a9f4d001caca51342001075469aff498"))
	check("Root", block.Root(), common.HexToHash("ef1552a40b7165c3cd773806b9e0c165b75356e0314bf0706f279c729f51e017"))
	check("Hash", block.Hash(), common.HexToHash("0a5843ac1cb04865017cb35a57b50b07084e5fcee39b5acadade33149f4fff9e"))
	check("Nonce", block.Nonce(), uint64(0xa13a5a8c8f2bb1c4))
	check("Time", block.Time(), uint64(1426516743))
	check("Size", block.Size(), common.StorageSize(len(blockEnc)))

	tx1 := NewTransaction(0, common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"), big.NewInt(10), 50000, big.NewInt(10), nil)
	tx1, _ = tx1.WithSignature(HomesteadSigner{}, common.Hex2Bytes("9bea4c4daac7c7c52e093e6a4c35dbbcf8856f1af7b059ba20253e70848d094f8a8fae537ce25ed8cb5af9adac3f141af69bd515bd2ba031522df09b97dd72b100"))
	check("len(Transactions)", len(block.Transactions()), 1)
	check("Transactions[0].Hash", block.Transactions()[0].Hash(), tx1.Hash())

	ourBlockEnc, err := rlp.EncodeToBytes(&block)
	if err != nil {
		t.Fatal("encode error: ", err)
	}
	if !bytes.Equal(ourBlockEnc, blockEnc) {
		t.Errorf("encoded block mismatch:\ngot:  %x\nwant: %x", ourBlockEnc, blockEnc)
	}
}

func TestUncleHash(t *testing.T) {
	uncles := make([]*Header, 0)
	h := CalcUncleHash(uncles)
	exp := common.HexToHash("1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347")
	if h != exp {
		t.Fatalf("empty uncle hash is wrong, got %x != %x", h, exp)
	}
}

var benchBuffer = bytes.NewBuffer(make([]byte, 0, 32000))

func BenchmarkEncodeBlock(b *testing.B) {
	block := makeBenchBlock()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		benchBuffer.Reset()
		if err := rlp.Encode(benchBuffer, block); err != nil {
			b.Fatal(err)
		}
	}
}

// testHasher is the helper tool for transaction/receipt list hashing.
// The original hasher is trie, in order to get rid of import cycle,
// use the testing hasher instead.
type testHasher struct {
	hasher hash.Hash
}

func newHasher() *testHasher {
	return &testHasher{hasher: sha3.NewLegacyKeccak256()}
}

func (h *testHasher) Reset() {
	h.hasher.Reset()
}

func (h *testHasher) Update(key, val []byte) {
	h.hasher.Write(key)
	h.hasher.Write(val)
}

func (h *testHasher) Hash() common.Hash {
	return common.BytesToHash(h.hasher.Sum(nil))
}

// TestVictionchainHeaderRoundtrip simulates receiving a header encoded by victionchain
// (which uses Go's default struct RLP encoding with 18 fields, no custom EncodeRLP).
// This test verifies that vic-geth's custom DecodeRLP correctly preserves all fields,
// especially the Extra field which must contain the 65-byte seal signature.
func TestVictionchainHeaderRoundtrip(t *testing.T) {
	// Build a header that looks like what victionchain would produce.
	// Extra = 32 bytes vanity + 65 bytes seal = 97 bytes total.
	vanity := make([]byte, 32)
	copy(vanity, []byte("viction-test"))
	seal := make([]byte, 65)
	for i := range seal {
		seal[i] = byte(i + 1) // non-zero to detect truncation
	}
	extra := append(vanity, seal...)

	// victionchain's Header struct layout (18 fields, no Posv bool):
	// ParentHash, UncleHash, Coinbase, Root, TxHash, ReceiptHash, Bloom,
	// Difficulty, Number, GasLimit, GasUsed, Time (*big.Int in victionchain!),
	// Extra, MixDigest, Nonce, Validators, Validator, Penalties
	//
	// We simulate victionchain's default struct RLP encoding by manually building
	// the RLP list with 18 fields.
	type victionHeader struct {
		ParentHash  common.Hash
		UncleHash   common.Hash
		Coinbase    common.Address
		Root        common.Hash
		TxHash      common.Hash
		ReceiptHash common.Hash
		Bloom       Bloom
		Difficulty  *big.Int
		Number      *big.Int
		GasLimit    uint64
		GasUsed     uint64
		Time        *big.Int // victionchain uses *big.Int
		Extra       []byte
		MixDigest   common.Hash
		Nonce       BlockNonce
		Validators  []byte // = NewAttestors in vic-geth
		Validator   []byte // = Attestor in vic-geth
		Penalties   []byte
	}

	vh := victionHeader{
		ParentHash:  common.HexToHash("0x1234"),
		UncleHash:   CalcUncleHash(nil),
		Root:        common.HexToHash("0xabcd"),
		TxHash:      common.HexToHash("0x5678"),
		ReceiptHash: common.HexToHash("0x9abc"),
		Difficulty:  big.NewInt(2),
		Number:      big.NewInt(941),
		GasLimit:    84000000,
		GasUsed:     0,
		Time:        big.NewInt(1715600000),
		Extra:       extra,
		Validators:  []byte{},
		Validator:   []byte{},
		Penalties:   []byte{},
	}

	// Encode using default struct RLP (no custom EncodeRLP) — this is what victionchain does.
	encoded, err := rlp.EncodeToBytes(&vh)
	if err != nil {
		t.Fatalf("encode victionchain header: %v", err)
	}

	// Decode using vic-geth's custom DecodeRLP
	var decoded Header
	if err := rlp.DecodeBytes(encoded, &decoded); err != nil {
		t.Fatalf("decode header: %v", err)
	}

	// Verify Extra is preserved
	if len(decoded.Extra) != 97 {
		t.Fatalf("Extra length mismatch: got %d, want 97", len(decoded.Extra))
	}
	if !bytes.Equal(decoded.Extra, extra) {
		t.Fatalf("Extra content mismatch:\ngot:  %x\nwant: %x", decoded.Extra, extra)
	}

	// Verify Posv flag is set
	if !decoded.Posv {
		t.Fatal("Posv should be true after decoding victionchain header")
	}

	// Verify other POSV fields
	if decoded.NewAttestors == nil {
		t.Fatal("NewAttestors should not be nil")
	}
	if decoded.Attestor == nil {
		t.Fatal("Attestor should not be nil")
	}
	if decoded.Penalties == nil {
		t.Fatal("Penalties should not be nil")
	}

	// Verify Number
	if decoded.Number.Uint64() != 941 {
		t.Fatalf("Number mismatch: got %d, want 941", decoded.Number.Uint64())
	}

	// Verify Time — victionchain sends *big.Int, vic-geth stores uint64
	if decoded.Time != 1715600000 {
		t.Fatalf("Time mismatch: got %d, want 1715600000", decoded.Time)
	}

	// Now test Hash() compatibility: vic-geth's Hash() should match victionchain's rlpHash
	vicGethHash := decoded.Hash()

	// Compute what victionchain would compute: rlpHash of the default struct encoding
	victionHash := rlpHash(&vh)

	if vicGethHash != victionHash {
		t.Fatalf("Hash mismatch between vic-geth and victionchain:\nvic-geth:     %s\nvictionchain: %s", vicGethHash.Hex(), victionHash.Hex())
	}

	t.Logf("Extra length: %d (correct)", len(decoded.Extra))
	t.Logf("Hash match: %s", vicGethHash.Hex())
}

// TestVictionchainBlockRoundtrip simulates receiving a full Block encoded by victionchain.
// The block is sent via NewBlockMsg as []interface{}{block, td}.
func TestVictionchainBlockRoundtrip(t *testing.T) {
	// Build a header as victionchain would — default struct RLP encoding with 18 fields.
	vanity := make([]byte, 32)
	copy(vanity, []byte("viction-test"))
	seal := make([]byte, 65)
	for i := range seal {
		seal[i] = byte(i + 1)
	}
	extra := append(vanity, seal...)

	// Simulate victionchain's Header struct (with *big.Int Time and Validators/Validator fields)
	type victionHeader struct {
		ParentHash  common.Hash
		UncleHash   common.Hash
		Coinbase    common.Address
		Root        common.Hash
		TxHash      common.Hash
		ReceiptHash common.Hash
		Bloom       Bloom
		Difficulty  *big.Int
		Number      *big.Int
		GasLimit    uint64
		GasUsed     uint64
		Time        *big.Int
		Extra       []byte
		MixDigest   common.Hash
		Nonce       BlockNonce
		Validators  []byte
		Validator   []byte
		Penalties   []byte
	}

	type victionExtblock struct {
		Header *victionHeader
		Txs    []*Transaction
		Uncles []*victionHeader
	}

	vh := &victionHeader{
		ParentHash:  common.HexToHash("0x1234"),
		UncleHash:   CalcUncleHash(nil),
		Root:        common.HexToHash("0xabcd"),
		TxHash:      EmptyRootHash,
		ReceiptHash: common.HexToHash("0x9abc"),
		Difficulty:  big.NewInt(2),
		Number:      big.NewInt(941),
		GasLimit:    84000000,
		GasUsed:     0,
		Time:        big.NewInt(1715600000),
		Extra:       extra,
		Validators:  []byte{},
		Validator:   []byte{},
		Penalties:   []byte{},
	}

	vBlock := &victionExtblock{
		Header: vh,
		Txs:    []*Transaction{},
		Uncles: []*victionHeader{},
	}

	// Encode the block as victionchain would
	encoded, err := rlp.EncodeToBytes(vBlock)
	if err != nil {
		t.Fatalf("encode victionchain block: %v", err)
	}

	// Decode as vic-geth would
	var block Block
	if err := rlp.DecodeBytes(encoded, &block); err != nil {
		t.Fatalf("decode block: %v", err)
	}

	// Verify Extra survived
	if len(block.Extra()) != 97 {
		t.Fatalf("Extra length mismatch: got %d, want 97", len(block.Extra()))
	}
	if !bytes.Equal(block.Extra(), extra) {
		t.Fatalf("Extra content mismatch:\ngot:  %x\nwant: %x", block.Extra(), extra)
	}

	// Verify header fields
	if block.NumberU64() != 941 {
		t.Fatalf("Number mismatch: got %d, want 941", block.NumberU64())
	}
	if block.Time() != 1715600000 {
		t.Fatalf("Time mismatch: got %d, want 1715600000", block.Time())
	}

	// Now simulate NewBlockMsg encoding: []interface{}{block, td}
	// victionchain sends: p2p.Send(p.rw, NewBlockMsg, []interface{}{block, td})
	td := big.NewInt(1000)
	msgPayload, err := rlp.EncodeToBytes([]interface{}{vBlock, td})
	if err != nil {
		t.Fatalf("encode NewBlockMsg payload: %v", err)
	}

	// Decode as vic-geth would in handler.go
	type newBlockData struct {
		Block *Block
		TD    *big.Int
	}
	var request newBlockData
	if err := rlp.DecodeBytes(msgPayload, &request); err != nil {
		t.Fatalf("decode NewBlockMsg: %v", err)
	}

	if len(request.Block.Extra()) != 97 {
		t.Fatalf("NewBlockMsg: Extra length mismatch: got %d, want 97", len(request.Block.Extra()))
	}
	if request.Block.NumberU64() != 941 {
		t.Fatalf("NewBlockMsg: Number mismatch: got %d, want 941", request.Block.NumberU64())
	}
	if !request.Block.header.Posv {
		t.Fatal("NewBlockMsg: Posv should be true")
	}

	t.Logf("Block Extra length: %d (correct)", len(request.Block.Extra()))
	t.Logf("Block Posv: %v", request.Block.header.Posv)
}

func makeBenchBlock() *Block {
	var (
		key, _   = crypto.GenerateKey()
		txs      = make([]*Transaction, 70)
		receipts = make([]*Receipt, len(txs))
		signer   = NewEIP155Signer(params.TestChainConfig.ChainID)
		uncles   = make([]*Header, 3)
	)
	header := &Header{
		Difficulty: math.BigPow(11, 11),
		Number:     math.BigPow(2, 9),
		GasLimit:   12345678,
		GasUsed:    1476322,
		Time:       9876543,
		Extra:      []byte("coolest block on chain"),
	}
	for i := range txs {
		amount := math.BigPow(2, int64(i))
		price := big.NewInt(300000)
		data := make([]byte, 100)
		tx := NewTransaction(uint64(i), common.Address{}, amount, 123457, price, data)
		signedTx, err := SignTx(tx, signer, key)
		if err != nil {
			panic(err)
		}
		txs[i] = signedTx
		receipts[i] = NewReceipt(make([]byte, 32), false, tx.Gas())
	}
	for i := range uncles {
		uncles[i] = &Header{
			Difficulty: math.BigPow(11, 11),
			Number:     math.BigPow(2, 9),
			GasLimit:   12345678,
			GasUsed:    1476322,
			Time:       9876543,
			Extra:      []byte("benchmark uncle"),
		}
	}
	return NewBlock(header, txs, uncles, receipts, newHasher())
}
