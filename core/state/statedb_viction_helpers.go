package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// GetLocDynamicArrAtElement is used to get the location of element inside dynamic array
func GetLocDynamicArrAtElement(slotHash common.Hash, index uint64, elementSize uint64) common.Hash {
	slotKecBig := crypto.Keccak256Hash(slotHash.Bytes()).Big()
	// arrBig = slotKecBig + index * elementSize
	arrBig := slotKecBig.Add(slotKecBig, new(big.Int).SetUint64(index*elementSize))
	return common.BigToHash(arrBig)
}

// GetLocMappingAtKey is used to get the location mapping at key
func GetLocMappingAtKey(locationIdx common.Hash, key []byte) common.Hash {
	req := append(common.LeftPadBytes(key, 32), locationIdx.Bytes()...)
	return crypto.Keccak256Hash(req)
}

// GetOwner returns the owner of a contract (slot 0).
// Used by the legacy TomoX order processor for relayer verification.
func (s *StateDB) GetOwner(addr common.Address) common.Address {
	return common.BytesToAddress(s.GetState(addr, common.Hash{}).Bytes())
}
