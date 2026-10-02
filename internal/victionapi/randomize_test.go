// Copyright 2014 The go-ethereum Authors
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

package victionapi

import (
	"bytes"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func GenerateRandomKey_Length(t *testing.T) {
	key := GenerateRandomKey()
	if len(key) != 32 {
		t.Fatalf("key length = %d, want 32", len(key))
	}
}

func GenerateRandomKey_Charset(t *testing.T) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"
	key := GenerateRandomKey()
	for i, b := range key {
		if !bytes.ContainsRune([]byte(charset), rune(b)) {
			t.Fatalf("key[%d] = %c, not in charset", i, b)
		}
	}
}

func GenerateRandomKey_Unique(t *testing.T) {
	k1 := GenerateRandomKey()
	time.Sleep(100 * time.Millisecond)
	k2 := GenerateRandomKey()
	if bytes.Equal(k1, k2) {
		t.Fatal("two generated keys should not be identical")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := GenerateRandomKey()
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
	key := GenerateRandomKey()
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

func CreateSetRandomizeSecretTransaction_Format(t *testing.T) {
	key := GenerateRandomKey()
	contract := common.HexToAddress("0x89")
	epoch := uint64(900)
	nonce := uint64(5)

	secret, err := GenerateRandomNumber(epoch, key)
	if err != nil {
		t.Fatalf("GenerateRandomNumber: %v", err)
	}
	tx := CreateSetRandomizeSecretTransaction(nonce, contract, secret)

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
	if !bytes.Equal(data[:4], common.Hex2Bytes("34d38600")) {
		t.Errorf("selector = %x, want %x", data[:4], common.Hex2Bytes("34d38600"))
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

func CreateSetRandomizeOpeningTransaction_Format(t *testing.T) {
	key := GenerateRandomKey()
	contract := common.HexToAddress("0x89")
	nonce := uint64(7)

	tx := CreateSetRandomizeOpeningTransaction(nonce, contract, key)

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
	if !bytes.Equal(data[:4], common.Hex2Bytes("e11f5ba2")) {
		t.Errorf("selector = %x, want %x", data[:4], common.Hex2Bytes("e11f5ba2"))
	}
	if !bytes.Equal(data[4:], key) {
		t.Error("opening data should contain the raw key")
	}
}

// --- End-to-end: secret tx can be decrypted with opening ---

func TestSecretAndOpening_EndToEnd(t *testing.T) {
	key := GenerateRandomKey()
	contract := common.HexToAddress("0x89")

	secret, err := GenerateRandomNumber(900, key)
	if err != nil {
		t.Fatalf("GenerateRandomNumber: %v", err)
	}
	secretTx := CreateSetRandomizeSecretTransaction(0, contract, secret)
	openingTx := CreateSetRandomizeOpeningTransaction(1, contract, key)

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
	key := GenerateRandomKey()
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
