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

package viction

import (
	"fmt"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	RandomizeKeyName = []byte("randomizeKey")
)

// Serialize `Randomize.setSecret“ function call data.
func CreateSetRandomizeSecretData(secret []byte) []byte {
	data := common.Hex2Bytes("34d38600") // setSecret(bytes32[])
	data = append(data, common.LeftPadBytes(big.NewInt(32).Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(big.NewInt(1).Bytes(), 32)...)
	data = append(data, common.LeftPadBytes([]byte(secret), 32)...)
	return data
}

// Create `Randomize.setSecret` transaction.
func CreateSetRandomizeSecretTransaction(nonce uint64, contractAddr common.Address, secret []byte) *types.Transaction {
	data := CreateSetRandomizeSecretData(secret)
	return types.NewTransaction(nonce, contractAddr, big.NewInt(0), 200000, big.NewInt(0), data)
}

// Serialize `Randomize.setOpening` function call data.
func CreateSetRandomizeOpeningData(key []byte) []byte {
	data := common.Hex2Bytes("e11f5ba2") // setOpening(bytes32)
	data = append(data, key...)
	return data
}

// Create `Randomize.setOpening` transaction.
func CreateSetRandomizeOpeningTransaction(nonce uint64, contractAddr common.Address, key []byte) *types.Transaction {
	data := CreateSetRandomizeOpeningData(key)
	return types.NewTransaction(nonce, contractAddr, big.NewInt(0), 200000, big.NewInt(0), data)
}

// Generate new 32-byte key for the randomize protocol.
func GenerateRandomKey() []byte {
	rand.Seed(time.Now().UnixNano())
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"
	b := make([]byte, 32)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return b
}

// Generate pseudo-random number and encrypt it using the given key.
func GenerateRandomNumber(max uint64, key []byte) ([]byte, error) {
	rand.Seed(time.Now().UnixNano())
	secretNumber := rand.Intn(int(max))
	encrypted, err := EncryptAesCfb(key, fmt.Sprintf("%d", secretNumber))
	if err != nil {
		return nil, fmt.Errorf("encrypt secret: %w", err)
	}
	return []byte(encrypted), nil
}

// Rerturn attestor indices from state.
func GetAttestorsFromState(
	validators []common.Address,
	victionConfig *params.VictionConfig,
	state *state.StateDB,
) ([]int64, error) {
	randomizes := []int64{}
	validatorCount := int64(len(validators))
	if validatorCount > 0 {
		for _, validator := range validators {
			random, err := GetSubmittedRandomOfValidator(validator, victionConfig, state)
			if err != nil {
				return nil, err
			}
			randomizes = append(randomizes, random)
		}
		attestors, err := GetAttestorsFromRandomize(randomizes, validatorCount)
		if err != nil {
			return nil, err
		}
		return attestors, nil
	}
	return nil, ErrNoValidator
}

// Get submitted random number of a given validator.
func GetSubmittedRandomOfValidator(
	validator common.Address,
	victionConfig *params.VictionConfig,
	state *state.StateDB,
) (int64, error) {
	randomizeContract := victionConfig.RandomizerContract
	if randomizeContract == (common.Address{}) {
		return -1, ErrNoContractAddress
	}

	secretsHash := state.VictionGetSecrets(randomizeContract, validator)
	openingHash := state.VictionGetSecretOpening(randomizeContract, validator)

	// Convert []common.Hash to [][32]byte
	secrets := make([][32]byte, len(secretsHash))
	for i, h := range secretsHash {
		secrets[i] = h
	}

	// Convert common.Hash to [32]byte
	opening := [32]byte(openingHash)

	result, err := DecryptRandomize(secrets, opening)
	if err != nil {
		return result, err
	}
	return result, nil
}

// Generate pseudo-random attestor indices based on randomizes.
func GetAttestorsFromRandomize(randomizes []int64, signersLen int64) ([]int64, error) {
	randomSeed := int64(0)
	for _, j := range randomizes {
		randomSeed += j
	}
	rand.Seed(randomSeed)

	randArray := GenerateSequence(0, 1, signersLen)
	attestorIndices := make([]int64, signersLen)
	attestorIndex := int64(0)
	for i := len(randArray) - 1; i >= 0; i-- {
		blockLength := len(randArray) - 1
		if blockLength <= 1 {
			blockLength = 1
		}
		randomIndex := int64(rand.Intn(blockLength))
		attestorIndex = randArray[randomIndex]
		randArray[randomIndex] = randArray[i]
		randArray[i] = attestorIndex
		randArray = append(randArray[:i], randArray[i+1:]...)
		attestorIndices[i] = attestorIndex
	}

	return attestorIndices, nil
}

// Crate new Randomize transaction and submit it to TxPool.
func SubmitRandomNumberTransaction(
	block *types.Block,
	snapshot *posv.Snapshot,
	config *params.ChainConfig,
	posvConfig *params.PosvConfig,
	victionConfig *params.VictionConfig,
	blockchain *core.BlockChain,
	chaindb ethdb.KeyValueStore,
	etherbase common.Address,
	wallet accounts.Wallet,
	txPool *core.TxPool,
) error {
	number := block.NumberU64()
	if !config.IsTIPRandomize(block.Number()) {
		return nil
	}
	if etherbase.IsZero() {
		return nil
	}
	if !IsEligibleValidator(etherbase, snapshot) {
		return nil
	}

	blockOfEpoch := number % posvConfig.Epoch
	commitPhase := victionConfig.IsRandomizerCommitPhase(blockOfEpoch)
	if commitPhase {
		exists, _ := chaindb.Has(RandomizeKeyName)
		if exists {
			// Already committed this epoch.
			return nil
		}

		nonce := NextNonce(etherbase, blockchain, txPool)
		key := GenerateRandomKey()
		secret, err := GenerateRandomNumber(posvConfig.Epoch, key)
		if err != nil {
			log.Error("[RandomNumber] Failed to generate RandomNumber", "number", number, "err", err)
			return err
		}
		transaction := CreateSetRandomizeSecretTransaction(nonce, victionConfig.RandomizerContract, secret)
		if err := SignAndSubmitTransaction(etherbase, wallet, transaction, config.ChainID, txPool, "RandomNumber"); err != nil {
			return err
		}
		if err := chaindb.Put(RandomizeKeyName, key); err != nil {
			log.Error("[RandomNumber] Failed to store RandomizeKey to database", "number", number, "err", err)
			return err
		}
		return nil
	}
	openingPhase := victionConfig.IsRandomizerOpeningPhase(blockOfEpoch)
	if openingPhase {
		key, err := chaindb.Get(RandomizeKeyName)
		if err != nil || len(key) == 0 {
			// Either already revealed or never committed in this epoch.
			return nil
		}

		nonce := NextNonce(etherbase, blockchain, txPool)
		transaction := CreateSetRandomizeOpeningTransaction(nonce, victionConfig.RandomizerContract, key)
		if err := SignAndSubmitTransaction(etherbase, wallet, transaction, config.ChainID, txPool, "RandomNumber"); err != nil {
			return err
		}
		if err := chaindb.Delete(RandomizeKeyName); err != nil {
			log.Error("[RandomNumber] Failed to delete RandomizeKey in database", "number", number, "err", err)
		}
	}
	return nil
}
