package posv

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// GetMasternodesFromCheckpointHeader extracts masternode addresses from the
// Extra field of a checkpoint header.
func GetMasternodesFromCheckpointHeader(checkpointHeader *types.Header) []common.Address {
	extra := checkpointHeader.Extra
	if len(extra) < extraVanity+extraSeal {
		return nil
	}
	masternodes := make([]common.Address, (len(extra)-extraVanity-extraSeal)/common.AddressLength)
	for i := 0; i < len(masternodes); i++ {
		copy(masternodes[i][:], extra[extraVanity+i*common.AddressLength:])
	}
	return masternodes
}

// CacheSigner filters signing transactions from a block and caches them.
// blockSignAddr is the ValidatorBlockSignContract address from chain config.
func (c *Posv) CacheSigner(hash common.Hash, txs []*types.Transaction, blockSignAddr common.Address) []*types.Transaction {
	signTxs := []*types.Transaction{}
	for _, tx := range txs {
		if tx.IsSigningTransaction(blockSignAddr) {
			signTxs = append(signTxs, tx)
		}
	}
	log.Debug("Save tx signers to cache", "hash", hash.String(), "len(txs)", len(signTxs))
	c.BlockSigners.Add(hash, signTxs)
	return signTxs
}
