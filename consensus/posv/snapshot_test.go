package posv

import (
	"bytes"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	lru "github.com/hashicorp/golang-lru"
)

// Test helper to create a header for testing
func makeHeader(number uint64, parentHash common.Hash, coinbase common.Address, nonce []byte) *types.Header {
	header := &types.Header{
		ParentHash: parentHash,
		Number:     big.NewInt(int64(number)),
		Time:       number * 15,
		GasLimit:   8000000,
		GasUsed:    0,
		Coinbase:   coinbase,
		Difficulty: big.NewInt(1),
		Extra:      make([]byte, ExtraVanity+ExtraSeal),
		MixDigest:  common.Hash{},
		UncleHash:  types.CalcUncleHash(nil),
	}
	copy(header.Nonce[:], nonce)
	return header
}

// Sign a header with the given private key
func signHeader(header *types.Header, key *ecdsa.PrivateKey) error {
	sig, err := crypto.Sign(SealHash(header).Bytes(), key)
	if err != nil {
		return err
	}
	copy(header.Extra[len(header.Extra)-ExtraSeal:], sig)
	return nil
}

// TestNewSnapshot tests the creation of a new snapshot
func TestNewSnapshot(t *testing.T) {
	config := &params.PosvConfig{
		Period: 2,
		Epoch:  900,
	}
	sigcache, _ := lru.NewARC(128)

	signers := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}

	snap := newSnapshot(config, sigcache, 0, common.Hash{}, signers)

	if snap.Number != 0 {
		t.Errorf("Expected number 0, got %d", snap.Number)
	}

	if len(snap.Signers) != len(signers) {
		t.Errorf("Expected %d signers, got %d", len(signers), len(snap.Signers))
	}

	for _, signer := range signers {
		if _, ok := snap.Signers[signer]; !ok {
			t.Errorf("Signer %s not found in snapshot", signer.Hex())
		}
	}

	if len(snap.Recents) != 0 {
		t.Errorf("Expected 0 recent signers, got %d", len(snap.Recents))
	}

	if len(snap.Votes) != 0 {
		t.Errorf("Expected 0 votes, got %d", len(snap.Votes))
	}
}

// TestSnapshotStore tests storing and loading a snapshot from the database
func TestSnapshotStore(t *testing.T) {
	config := &params.PosvConfig{
		Period: 2,
		Epoch:  900,
	}
	sigcache, _ := lru.NewARC(128)
	db := rawdb.NewMemoryDatabase()

	signers := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}

	hash := common.HexToHash("0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")
	snap := newSnapshot(config, sigcache, 100, hash, signers)

	// Add some recents
	snap.Recents[95] = signers[0]
	snap.Recents[98] = signers[1]

	// Store the snapshot
	if err := snap.store(db); err != nil {
		t.Fatalf("Failed to store snapshot: %v", err)
	}

	// Load the snapshot
	loaded, err := loadSnapshot(config, sigcache, db, hash)
	if err != nil {
		t.Fatalf("Failed to load snapshot: %v", err)
	}

	// Verify the loaded snapshot
	if loaded.Number != snap.Number {
		t.Errorf("Expected number %d, got %d", snap.Number, loaded.Number)
	}

	if loaded.Hash != snap.Hash {
		t.Errorf("Expected hash %s, got %s", snap.Hash.Hex(), loaded.Hash.Hex())
	}

	if len(loaded.Signers) != len(snap.Signers) {
		t.Errorf("Expected %d signers, got %d", len(snap.Signers), len(loaded.Signers))
	}

	if len(loaded.Recents) != len(snap.Recents) {
		t.Errorf("Expected %d recents, got %d", len(snap.Recents), len(loaded.Recents))
	}
}

// TestSnapshotCopy tests the snapshot copy method
func TestSnapshotCopy(t *testing.T) {
	config := &params.PosvConfig{
		Period: 2,
		Epoch:  900,
	}
	sigcache, _ := lru.NewARC(128)

	signers := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}

	snap := newSnapshot(config, sigcache, 100, common.Hash{}, signers)
	snap.Recents[95] = signers[0]
	snap.Recents[98] = signers[1]
	snap.Votes = append(snap.Votes, &clique.Vote{
		Signer:    signers[0],
		Block:     90,
		Address:   common.HexToAddress("0x4444444444444444444444444444444444444444"),
		Authorize: true,
	})

	// Create a copy
	copy := snap.copy()

	// Verify the copy
	if copy.Number != snap.Number {
		t.Errorf("Expected number %d, got %d", snap.Number, copy.Number)
	}

	if len(copy.Signers) != len(snap.Signers) {
		t.Errorf("Expected %d signers, got %d", len(snap.Signers), len(copy.Signers))
	}

	if len(copy.Recents) != len(snap.Recents) {
		t.Errorf("Expected %d recents, got %d", len(snap.Recents), len(copy.Recents))
	}

	if len(copy.Votes) != len(snap.Votes) {
		t.Errorf("Expected %d votes, got %d", len(snap.Votes), len(copy.Votes))
	}

	// Modify the copy and ensure the original is unchanged
	copy.Recents[100] = signers[0]
	if len(snap.Recents) == len(copy.Recents) {
		t.Error("Modifying copy affected the original snapshot")
	}
}

// TestValidVote tests the validVote method
func TestValidVote(t *testing.T) {
	config := &params.PosvConfig{
		Period: 2,
		Epoch:  900,
	}
	sigcache, _ := lru.NewARC(128)

	signer1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	signer2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	nonSigner := common.HexToAddress("0x3333333333333333333333333333333333333333")

	snap := newSnapshot(config, sigcache, 0, common.Hash{}, []common.Address{signer1, signer2})

	// Valid votes
	if !snap.validVote(nonSigner, true) {
		t.Error("Should allow authorizing a non-signer")
	}

	if !snap.validVote(signer1, false) {
		t.Error("Should allow deauthorizing a signer")
	}

	// Invalid votes
	if snap.validVote(signer1, true) {
		t.Error("Should not allow authorizing an existing signer")
	}

	if snap.validVote(nonSigner, false) {
		t.Error("Should not allow deauthorizing a non-signer")
	}
}

// TestCastAndUncast tests the cast and uncast methods
func TestCastAndUncast(t *testing.T) {
	config := &params.PosvConfig{
		Period: 2,
		Epoch:  900,
	}
	sigcache, _ := lru.NewARC(128)

	signer1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	newAddress := common.HexToAddress("0x3333333333333333333333333333333333333333")

	snap := newSnapshot(config, sigcache, 0, common.Hash{}, []common.Address{signer1})

	// Cast a vote to add a new signer
	if !snap.cast(newAddress, true) {
		t.Error("Failed to cast vote")
	}

	if tally, ok := snap.Tally[newAddress]; !ok {
		t.Error("Vote not recorded in tally")
	} else if tally.Votes != 1 || !tally.Authorize {
		t.Errorf("Expected 1 authorize vote, got %d votes, authorize=%v", tally.Votes, tally.Authorize)
	}

	// Cast another vote
	if !snap.cast(newAddress, true) {
		t.Error("Failed to cast second vote")
	}

	if tally := snap.Tally[newAddress]; tally.Votes != 2 {
		t.Errorf("Expected 2 votes, got %d", tally.Votes)
	}

	// Uncast a vote
	if !snap.uncast(newAddress, true) {
		t.Error("Failed to uncast vote")
	}

	if tally := snap.Tally[newAddress]; tally.Votes != 1 {
		t.Errorf("Expected 1 vote after uncast, got %d", tally.Votes)
	}

	// Uncast the last vote
	if !snap.uncast(newAddress, true) {
		t.Error("Failed to uncast last vote")
	}

	if _, ok := snap.Tally[newAddress]; ok {
		t.Error("Expected tally to be removed after all votes uncast")
	}

	// Try to uncast with wrong authorization
	snap.cast(newAddress, true)
	if snap.uncast(newAddress, false) {
		t.Error("Should not allow uncasting with wrong authorization")
	}
}

// TestGetSigners tests the GetSigners method
func TestGetSigners(t *testing.T) {
	config := &params.PosvConfig{
		Period: 2,
		Epoch:  900,
	}
	sigcache, _ := lru.NewARC(128)

	signers := []common.Address{
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
	}

	snap := newSnapshot(config, sigcache, 0, common.Hash{}, signers)

	result := snap.GetSigners()

	// Should return sorted addresses
	if len(result) != len(signers) {
		t.Errorf("Expected %d signers, got %d", len(signers), len(result))
	}

	// Verify sorted order
	for i := 0; i < len(result)-1; i++ {
		if bytes.Compare(result[i][:], result[i+1][:]) >= 0 {
			t.Error("Signers not properly sorted")
		}
	}
}

// TestInturn tests the inturn method
func TestInturn(t *testing.T) {
	config := &params.PosvConfig{
		Period: 2,
		Epoch:  900,
	}
	sigcache, _ := lru.NewARC(128)

	signers := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}

	snap := newSnapshot(config, sigcache, 0, common.Hash{}, signers)

	// Get sorted signers
	sortedSigners := snap.GetSigners()

	// Test inturn for different block numbers
	for i := uint64(0); i < 10; i++ {
		expectedSigner := sortedSigners[i%uint64(len(sortedSigners))]
		if !snap.inturn(i, expectedSigner) {
			t.Errorf("Block %d: Expected signer %s to be inturn", i, expectedSigner.Hex())
		}

		// Test that other signers are not inturn
		for _, signer := range sortedSigners {
			if signer != expectedSigner && snap.inturn(i, signer) {
				t.Errorf("Block %d: Signer %s should not be inturn", i, signer.Hex())
			}
		}
	}
}

// TestSnapshotApply tests applying headers to a snapshot
func TestSnapshotApply(t *testing.T) {
	config := &params.PosvConfig{
		Period: 2,
		Epoch:  900,
	}
	sigcache, _ := lru.NewARC(128)

	// Create test signers
	key1, _ := crypto.GenerateKey()
	key2, _ := crypto.GenerateKey()

	signer1 := crypto.PubkeyToAddress(key1.PublicKey)
	signer2 := crypto.PubkeyToAddress(key2.PublicKey)

	// Create initial snapshot
	snap := newSnapshot(config, sigcache, 0, common.Hash{}, []common.Address{signer1, signer2})

	// Create a chain of headers
	var headers []*types.Header
	parentHash := common.Hash{}

	for i := uint64(1); i <= 5; i++ {
		key := key1
		if i%2 == 0 {
			key = key2
		}

		header := makeHeader(i, parentHash, common.Address{}, nonceDropVote)
		if err := signHeader(header, key); err != nil {
			t.Fatalf("Failed to sign header: %v", err)
		}

		headers = append(headers, header)
		parentHash = header.Hash()
	}

	// Apply headers
	newSnap, err := snap.apply(headers)
	if err != nil {
		t.Fatalf("Failed to apply headers: %v", err)
	}

	// Verify the new snapshot
	if newSnap.Number != snap.Number+uint64(len(headers)) {
		t.Errorf("Expected number %d, got %d", snap.Number+uint64(len(headers)), newSnap.Number)
	}

	if newSnap.Hash != headers[len(headers)-1].Hash() {
		t.Error("Snapshot hash doesn't match last header hash")
	}

	// Verify recents are tracked
	if len(newSnap.Recents) == 0 {
		t.Error("Expected recent signers to be tracked")
	}
}

// TestSnapshotApplyCheckpoint tests behavior at checkpoint blocks
func TestSnapshotApplyCheckpoint(t *testing.T) {
	config := &params.PosvConfig{
		Period: 2,
		Epoch:  10, // Small epoch for testing
	}
	sigcache, _ := lru.NewARC(128)

	// Create test signers
	key1, _ := crypto.GenerateKey()
	signer1 := crypto.PubkeyToAddress(key1.PublicKey)

	// Create initial snapshot
	snap := newSnapshot(config, sigcache, 5, common.Hash{}, []common.Address{signer1})

	// Add some votes
	snap.Votes = append(snap.Votes, &clique.Vote{
		Signer:    signer1,
		Block:     5,
		Address:   common.HexToAddress("0x4444444444444444444444444444444444444444"),
		Authorize: true,
	})
	snap.Tally[common.HexToAddress("0x4444444444444444444444444444444444444444")] = clique.Tally{
		Authorize: true,
		Votes:     1,
	}

	// Create headers up to checkpoint
	var headers []*types.Header
	parentHash := common.Hash{}

	for i := uint64(6); i <= 10; i++ {
		header := makeHeader(i, parentHash, common.Address{}, nonceDropVote)
		if err := signHeader(header, key1); err != nil {
			t.Fatalf("Failed to sign header: %v", err)
		}

		headers = append(headers, header)
		parentHash = header.Hash()
	}

	// Apply headers
	newSnap, err := snap.apply(headers)
	if err != nil {
		t.Fatalf("Failed to apply headers: %v", err)
	}

	// At checkpoint, votes and tally should be reset
	if len(newSnap.Votes) != 0 {
		t.Errorf("Expected votes to be cleared at checkpoint, got %d votes", len(newSnap.Votes))
	}

	if len(newSnap.Tally) != 0 {
		t.Errorf("Expected tally to be cleared at checkpoint, got %d entries", len(newSnap.Tally))
	}
}

// TestSnapshotApplyInvalidChain tests error handling for invalid header chains
func TestSnapshotApplyInvalidChain(t *testing.T) {
	config := &params.PosvConfig{
		Period: 2,
		Epoch:  900,
	}
	sigcache, _ := lru.NewARC(128)

	key1, _ := crypto.GenerateKey()
	signer1 := crypto.PubkeyToAddress(key1.PublicKey)

	snap := newSnapshot(config, sigcache, 10, common.Hash{}, []common.Address{signer1})

	// Test 1: Non-contiguous headers
	headers := []*types.Header{
		makeHeader(11, common.Hash{}, common.Address{}, nonceDropVote),
		makeHeader(13, common.Hash{}, common.Address{}, nonceDropVote), // Skip block 12
	}

	_, err := snap.apply(headers)
	if err != errInvalidVotingChain {
		t.Errorf("Expected errInvalidVotingChain for non-contiguous headers, got: %v", err)
	}

	// Test 2: Headers starting from wrong number
	headers = []*types.Header{
		makeHeader(15, common.Hash{}, common.Address{}, nonceDropVote),
	}

	_, err = snap.apply(headers)
	if err != errInvalidVotingChain {
		t.Errorf("Expected errInvalidVotingChain for wrong starting number, got: %v", err)
	}
}
