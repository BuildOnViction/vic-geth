package state

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
)

// GetLocDynamicArrAtElement is used to get the location of element inside dynamic array
func GetLocDynamicArrAtElement(locationIdx common.Hash, eId uint64) common.Hash {
	b1 := locationIdx.Bytes()
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(b1)
	res1 := hasher.Sum(nil)

	sumB := make([]byte, 8)
	binary.BigEndian.PutUint64(sumB, eId)

	res := common.BytesToHash(res1)
	sumHash := common.BytesToHash(sumB)

	finalBig := new(big.Int).Add(res.Big(), sumHash.Big())
	return common.BigToHash(finalBig)
}

// GetLocMappingAtKey is used to get the location mapping at key
func GetLocMappingAtKey(locationIdx common.Hash, key []byte) common.Hash {
	req := append(common.LeftPadBytes(key, 32), locationIdx.Bytes()...)
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(req)
	return common.BytesToHash(hasher.Sum(nil))
}
