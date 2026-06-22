// Copyright (c) 2026 Viction
// Randomize transaction construction for POSV validators.
//
// Validators participate in a two-phase commit/reveal scheme each epoch:
//   Phase 1 (Commit): At blocks [CommitNthBlock, RevealNthBlock) within the epoch,
//                     generate a random key, encrypt a random number with it,
//                     and submit setSecret(bytes32[]) to the Randomize contract.
//   Phase 2 (Reveal): At blocks [RevealNthBlock, FinaleNthBlock] within the epoch,
//                     submit setOpening(bytes32) with the key used in Phase 1.
//
// The decrypted random values from all validators are summed to seed a
// deterministic shuffle that assigns attestor (M2) roles for the next epoch.

package viction

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	mathrand "math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// Method selectors for the Randomize contract (0x89).
var (
	// setSecret(bytes32[]) — 0x34d38600
	SetSecretSelector = common.Hex2Bytes("34d38600")
	// setOpening(bytes32) — 0xe11f5ba2
	SetOpeningSelector = common.Hex2Bytes("e11f5ba2")
)

// RandomizeKeyName is the database key used to persist the random key between
// the commit and reveal phases within a single epoch.
var RandomizeKeyName = []byte("randomizeKey")

// ShouldSendSecret returns true if the current block position within the epoch
// falls in the commit phase: [CommitNthBlock, RevealNthBlock).
func ShouldSendSecret(vicConfig *params.VictionConfig, blockInEpoch uint64) bool {
	return blockInEpoch > 0 &&
		blockInEpoch >= vicConfig.RandomizerCommitNthBlock &&
		blockInEpoch < vicConfig.RandomizerRevealNthBlock
}

// ShouldSendOpening returns true if the current block position within the epoch
// falls in the reveal phase: [RevealNthBlock, FinaleNthBlock].
func ShouldSendOpening(vicConfig *params.VictionConfig, blockInEpoch uint64) bool {
	return blockInEpoch > 0 &&
		blockInEpoch >= vicConfig.RandomizerRevealNthBlock &&
		blockInEpoch <= vicConfig.RandomizerFinaleNthBlock
}

// GenerateRandomizeKey creates a 32-byte random key for AES encryption.
// Uses alphanumeric characters to match victionchain's RandStringByte(32).
func GenerateRandomizeKey() []byte {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"
	mathrand.Seed(time.Now().UnixNano())
	b := make([]byte, 32)
	for i := range b {
		b[i] = charset[mathrand.Intn(len(charset))]
	}
	return b
}

// BuildSecretTx constructs a setSecret(bytes32[]) transaction for the Randomize
// contract. It generates a random number in [0, epoch), encrypts it with the
// provided key using AES-CFB, and encodes it as a single-element bytes32 array.
func BuildSecretTx(nonce uint64, randomizeContract common.Address, epoch uint64, key []byte) (*types.Transaction, error) {
	mathrand.Seed(time.Now().UnixNano())
	secretNumber := mathrand.Intn(int(epoch))

	encrypted, err := EncryptAesCfb(key, fmt.Sprintf("%d", secretNumber))
	if err != nil {
		return nil, fmt.Errorf("[Viction] encrypt secret: %w", err)
	}

	// ABI encoding: setSecret(bytes32[])
	// Layout: selector(4) + offset(32) + length(32) + element(32)
	const slotSize = 32
	data := make([]byte, 0, 4+slotSize*3)
	data = append(data, SetSecretSelector...)
	data = append(data, common.LeftPadBytes(big.NewInt(int64(slotSize)).Bytes(), slotSize)...) // offset to array data
	data = append(data, common.LeftPadBytes(big.NewInt(1).Bytes(), slotSize)...)               // array length = 1
	data = append(data, common.LeftPadBytes([]byte(encrypted), slotSize)...)                   // encrypted secret

	tx := types.NewTransaction(nonce, randomizeContract, big.NewInt(0), 200000, big.NewInt(0), data)
	return tx, nil
}

// BuildOpeningTx constructs a setOpening(bytes32) transaction for the Randomize
// contract. The opening is the raw AES key used to encrypt the secret.
func BuildOpeningTx(nonce uint64, randomizeContract common.Address, key []byte) *types.Transaction {
	data := make([]byte, 0, 4+32)
	data = append(data, SetOpeningSelector...)
	data = append(data, key...)

	return types.NewTransaction(nonce, randomizeContract, big.NewInt(0), 200000, big.NewInt(0), data)
}

// EncryptAesCfb encrypts plaintext using AES-CFB mode with a random IV.
// Returns a base64url-encoded string: IV || ciphertext.
func EncryptAesCfb(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plain := []byte(plaintext)
	ciphertext := make([]byte, aes.BlockSize+len(plain))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plain)

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}
