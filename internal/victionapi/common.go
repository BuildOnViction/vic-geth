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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// ErrInvalidAttestorList is when the attestors list are not conformed to the consensus rules.
	ErrInvalidAttestorList = errors.New("invalid attestor list")

	// ErrNilHeader is when header is nil.
	ErrNilHeader = errors.New("nil header")

	// ErrNoContractAddress is when the contract address is not set in the config.
	ErrNoContractAddress = errors.New("contract address is not set")

	// ErrNoValidator is when the list of validator is empty.
	ErrNoValidator = errors.New("no validators exist")
)

// Decrypt encrypted data using AES CFB mode.
func DecryptAesCfb(key []byte, cryptoText string) (string, error) {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return string(ciphertext), nil
}

// Decrypt randomize using secret and opening pair.
func DecryptRandomize(secrets [][32]byte, opening [32]byte) (int64, error) {
	var random int64
	if len(secrets) > 0 {
		for _, secret := range secrets {
			trimSecret := bytes.TrimLeft(secret[:], "\x00")
			decryptSecret, err := DecryptAesCfb(opening[:], string(trimSecret))
			if err != nil {
				return 0, err
			}
			intNumber, err := strconv.ParseInt(decryptSecret, 10, 64)
			if err == nil {
				random = intNumber
			}
		}
	}

	return random, nil
}

// Encrypt data using AES CFB mode.
func EncryptAesCfb(key []byte, text string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	plaintext := []byte(text)
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// Generate a dynamic array from *start*, increase by *step* unit by *repeat* times.
func GenerateSequence(start, step, repeat int64) []int64 {
	s := make([]int64, repeat)
	v := start
	for i := range s {
		s[i] = v
		v += step
	}

	return s
}

// Check if given addr is eligible signer for given snapshot.
func IsEligibleValidator(addr common.Address, snapshot *posv.Snapshot) bool {
	_, ok := snapshot.Signers[addr]
	return ok
}

// Return available nonce for the given address, with txPool awareness.
func NextNonce(addr common.Address, blockchain *core.BlockChain, txPool *core.TxPool) uint64 {
	statedb, err := blockchain.State()
	if err != nil {
		return 0
	}
	nonce := statedb.GetNonce(addr)
	pendingTxs, _ := txPool.Pending()
	nonce += uint64(len(pendingTxs[addr]))
	return nonce
}

// Sign and submit given transaction to local txPool.
func SignAndSubmitTransaction(etherbase common.Address, wallet accounts.Wallet, transaction *types.Transaction, chainID *big.Int, txPool *core.TxPool, label string) error {
	signedTransaction, err := wallet.SignTx(accounts.Account{Address: etherbase}, transaction, chainID)
	if err != nil {
		return err
	}
	if err := txPool.AddLocal(signedTransaction); err != nil {
		if err == core.ErrReplaceUnderpriced || err == core.ErrAlreadyKnown {
			log.Info(fmt.Sprintf("[%s] Transaction is duplicated", label), "txHash", signedTransaction.Hash(), "etherbase", etherbase, "nonce", signedTransaction.Nonce())
			return nil
		}
		log.Warn(fmt.Sprintf("[%s] Failed to submit transaction", label), "txHash", signedTransaction.Hash(), "etherbase", etherbase, "nonce", signedTransaction.Nonce(), "err", err)
		return err
	}
	log.Info(fmt.Sprintf("[%s] Submitted transaction", label), "txHash", signedTransaction.Hash(), "etherbase", etherbase, "nonce", signedTransaction.Nonce())
	return nil
}
