// Copyright (c) 2026 Viction
package viction

import (
	"bytes"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

var testVicConfig = &params.VictionConfig{
	RandomizerContract:       common.HexToAddress("0x89"),
	RandomizerCommitNthBlock: 800,
	RandomizerRevealNthBlock: 850,
	RandomizerFinaleNthBlock: 900,
}

// --- ShouldSendSecret / ShouldSendOpening ---

func TestShouldSendSecret(t *testing.T) {
	tests := []struct {
		blockInEpoch uint64
		want         bool
	}{
		{0, false},   // epoch boundary itself
		{1, false},   // too early
		{799, false}, // one before commit start
		{800, true},  // commit start
		{825, true},  // mid commit
		{849, true},  // last commit block
		{850, false}, // reveal start (no longer commit)
		{900, false}, // finale
	}
	for _, tt := range tests {
		got := ShouldSendSecret(testVicConfig, tt.blockInEpoch)
		if got != tt.want {
			t.Errorf("ShouldSendSecret(blockInEpoch=%d) = %v, want %v", tt.blockInEpoch, got, tt.want)
		}
	}
}

func TestShouldSendOpening(t *testing.T) {
	tests := []struct {
		blockInEpoch uint64
		want         bool
	}{
		{0, false},
		{800, false}, // commit phase
		{849, false}, // last commit block
		{850, true},  // reveal start
		{875, true},  // mid reveal
		{900, true},  // finale (inclusive)
		{901, false}, // past finale
	}
	for _, tt := range tests {
		got := ShouldSendOpening(testVicConfig, tt.blockInEpoch)
		if got != tt.want {
			t.Errorf("ShouldSendOpening(blockInEpoch=%d) = %v, want %v", tt.blockInEpoch, got, tt.want)
		}
	}
}

// --- GenerateRandomizeKey ---

func TestGenerateRandomizeKey_Length(t *testing.T) {
	key := GenerateRandomizeKey()
	if len(key) != 32 {
		t.Fatalf("key length = %d, want 32", len(key))
	}
}

func TestGenerateRandomizeKey_Charset(t *testing.T) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"
	key := GenerateRandomizeKey()
	for i, b := range key {
		if !bytes.ContainsRune([]byte(charset), rune(b)) {
			t.Fatalf("key[%d] = %c, not in charset", i, b)
		}
	}
}

func TestGenerateRandomizeKey_Unique(t *testing.T) {
	k1 := GenerateRandomizeKey()
	k2 := GenerateRandomizeKey()
	if bytes.Equal(k1, k2) {
		t.Fatal("two generated keys should not be identical")
	}
}

// --- EncryptAesCfb / DecryptAesCfb round-trip ---

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := GenerateRandomizeKey()
	plaintext := "42"

	encrypted, err := EncryptAesCfb(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if encrypted == plaintext {
		t.Fatal("encrypted should differ from plaintext")
	}

	decrypted, err := DecryptAesCfb(key, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecryptRoundTrip_LargeNumber(t *testing.T) {
	key := GenerateRandomizeKey()
	plaintext := "899" // max for epoch=900

	encrypted, err := EncryptAesCfb(key, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptAesCfb(key, encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

// --- BuildSecretTx ---

func TestBuildSecretTx_Format(t *testing.T) {
	key := GenerateRandomizeKey()
	contract := common.HexToAddress("0x89")
	epoch := uint64(900)
	nonce := uint64(5)

	tx, err := BuildSecretTx(nonce, contract, epoch, key)
	if err != nil {
		t.Fatalf("BuildSecretTx: %v", err)
	}

	// Basic tx properties
	if tx.Nonce() != nonce {
		t.Errorf("nonce = %d, want %d", tx.Nonce(), nonce)
	}
	if *tx.To() != contract {
		t.Errorf("to = %v, want %v", tx.To(), contract)
	}
	if tx.Value().Sign() != 0 {
		t.Error("value should be 0")
	}
	if tx.Gas() != 200000 {
		t.Errorf("gas = %d, want 200000", tx.Gas())
	}
	if tx.GasPrice().Sign() != 0 {
		t.Error("gasPrice should be 0")
	}

	// Data format: selector(4) + offset(32) + length(32) + element(32) = 100 bytes
	data := tx.Data()
	if len(data) != 4+32+32+32 {
		t.Fatalf("data length = %d, want 100", len(data))
	}

	// Check selector
	if !bytes.Equal(data[:4], SetSecretSelector) {
		t.Errorf("selector = %x, want %x", data[:4], SetSecretSelector)
	}

	// Check offset = 32
	offset := new(big.Int).SetBytes(data[4:36])
	if offset.Int64() != 32 {
		t.Errorf("offset = %d, want 32", offset.Int64())
	}

	// Check array length = 1
	arrLen := new(big.Int).SetBytes(data[36:68])
	if arrLen.Int64() != 1 {
		t.Errorf("array length = %d, want 1", arrLen.Int64())
	}

	// The encrypted element should be decryptable
	element := bytes.TrimLeft(data[68:100], "\x00")
	decrypted, err := DecryptAesCfb(key, string(element))
	if err != nil {
		t.Fatalf("decrypt element: %v", err)
	}
	num, err := strconv.ParseInt(decrypted, 10, 64)
	if err != nil {
		t.Fatalf("parse decrypted: %v", err)
	}
	if num < 0 || num >= int64(epoch) {
		t.Errorf("secret number = %d, should be in [0, %d)", num, epoch)
	}
}

// --- BuildOpeningTx ---

func TestBuildOpeningTx_Format(t *testing.T) {
	key := GenerateRandomizeKey()
	contract := common.HexToAddress("0x89")
	nonce := uint64(7)

	tx := BuildOpeningTx(nonce, contract, key)

	if tx.Nonce() != nonce {
		t.Errorf("nonce = %d, want %d", tx.Nonce(), nonce)
	}
	if *tx.To() != contract {
		t.Errorf("to = %v, want %v", tx.To(), contract)
	}
	if tx.Value().Sign() != 0 {
		t.Error("value should be 0")
	}
	if tx.Gas() != 200000 {
		t.Errorf("gas = %d, want 200000", tx.Gas())
	}
	if tx.GasPrice().Sign() != 0 {
		t.Error("gasPrice should be 0")
	}

	// Data: selector(4) + key(32) = 36 bytes
	data := tx.Data()
	if len(data) != 4+32 {
		t.Fatalf("data length = %d, want 36", len(data))
	}
	if !bytes.Equal(data[:4], SetOpeningSelector) {
		t.Errorf("selector = %x, want %x", data[:4], SetOpeningSelector)
	}
	if !bytes.Equal(data[4:], key) {
		t.Error("opening data should contain the raw key")
	}
}

// --- End-to-end: secret tx can be decrypted with opening ---

func TestSecretAndOpening_EndToEnd(t *testing.T) {
	key := GenerateRandomizeKey()
	contract := common.HexToAddress("0x89")

	secretTx, err := BuildSecretTx(0, contract, 900, key)
	if err != nil {
		t.Fatalf("BuildSecretTx: %v", err)
	}
	openingTx := BuildOpeningTx(1, contract, key)

	// Extract the key from opening tx data
	openingKey := openingTx.Data()[4:]

	// Extract the encrypted secret from secret tx data
	secretData := secretTx.Data()
	encryptedElement := bytes.TrimLeft(secretData[68:100], "\x00")

	// Decrypt using the opening key
	decrypted, err := DecryptAesCfb(openingKey, string(encryptedElement))
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	num, err := strconv.ParseInt(decrypted, 10, 64)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if num < 0 || num >= 900 {
		t.Errorf("decrypted value = %d, want in [0, 900)", num)
	}
}

// --- GetAttestors integration: verify decrypt matches shuffle input ---

func TestDecryptRandomize_MatchesEncrypt(t *testing.T) {
	key := GenerateRandomizeKey()
	plaintext := "567"

	encrypted, err := EncryptAesCfb(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate what GetRandomizeOfValidator does: decrypt using secrets/opening
	secret := common.LeftPadBytes([]byte(encrypted), 32)
	var s32 [32]byte
	copy(s32[:], secret)
	secrets := [][32]byte{s32}
	opening := [32]byte{}
	copy(opening[:], key)

	result, err := DecryptRandomize(secrets, opening)
	if err != nil {
		t.Fatalf("DecryptRandomize: %v", err)
	}
	if result != 567 {
		t.Errorf("result = %d, want 567", result)
	}
}
