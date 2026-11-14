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
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posv"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	lru "github.com/hashicorp/golang-lru"
)

// Attempt to attest the given block. New block will only be returned if the attestor's signature is added to the block.
func AttestBlock(
	block *types.Block, checkpointHeader *types.Header,
	config *params.ChainConfig, posvConfig *params.PosvConfig,
	etherbase common.Address, wallet accounts.Wallet,
) (*types.Block, error) {
	header := block.Header()
	number := header.Number.Uint64()
	if number <= posvConfig.Epoch {
		return block, nil
	}
	// Only accept non-attested block.
	if len(header.Attestor) == posv.ExtraSeal {
		return nil, nil
	}
	sigCache, err := lru.NewARC(128)
	if err != nil {
		return nil, nil
	}
	creator, err := posv.Ecrecover(header, sigCache)
	if err != nil {
		return nil, nil
	}
	valAttPairs, _, err := GetCreatorAttestorPairsFromCheckpointHeader(config, posvConfig, header, checkpointHeader)
	if err != nil && err != ErrNoValidator {
		return nil, nil
	}
	assigned, ok := valAttPairs[creator]
	if !ok || etherbase != assigned {
		return nil, nil
	}
	sig, err := wallet.SignData(accounts.Account{Address: etherbase}, accounts.MimetypePosv, posv.PosvRLP(header))
	if err != nil {
		return nil, err
	}
	attestedHeader := types.CopyHeader(header)
	attestedHeader.Attestor = make([]byte, len(sig))
	copy(attestedHeader.Attestor, sig)
	attestedBlock := block.WithSeal(attestedHeader)
	attestedBlock.ReceivedAt = block.ReceivedAt // preserve for propagation-latency metrics
	return attestedBlock, nil
}
