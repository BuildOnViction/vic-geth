package viction

import (
	"math/rand"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/randomize/contract"
	"github.com/ethereum/go-ethereum/params"
)

func GetAttestors(vicConfig *params.VictionConfig, validators []common.Address, client bind.ContractBackend) ([]int64, error) {
	randomizes := []int64{}
	validatorCount := int64(len(validators))
	if validatorCount > 0 {
		for _, validator := range validators {
			random, err := GetRandomizeOfValidator(vicConfig, validator, client)
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

func GetRandomizeOfValidator(vicConfig *params.VictionConfig, validator common.Address, client bind.ContractBackend) (int64, error) {
	randomizeContract, err := contract.NewRandomize(vicConfig.RandomizerContract, client)
	if err != nil {
		return -1, err
	}

	opts := new(bind.CallOpts)
	secrets, err := randomizeContract.GetSecret(opts, validator)
	if err != nil {
		return -1, err
	}
	opening, err := randomizeContract.GetOpening(opts, validator)
	if err != nil {
		return -1, err
	}

	return DecryptRandomize(secrets, opening)
}

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
