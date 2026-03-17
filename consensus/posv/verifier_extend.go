// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
// Copyright 2025 The Viction Authors
// (modifications)
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

package posv

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// Enhance verifyHeader by caching the result to speed up repeated verifications.
func (c *Posv) verifyHeaderWithCache(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	_, ok := c.verifiedBlocks.Get(header.Hash())
	if ok {
		return nil
	}

	err := c.verifyHeader(chain, header, parents)
	if err == nil {
		c.verifiedBlocks.Add(header.Hash(), true)
	}
	return err
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements.
func (c *Posv) verifySeal(chainH consensus.ChainHeaderReader, snap *Snapshot, header *types.Header, parents []*types.Header, logger log.Logger) error {
	chain := chainH.(consensus.ChainReader)

	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	// Resolve the authorization key and check against signers
	validators, err := c.backend.PosvGetValidators(c.chainConfig.Viction, header, logger)
	if err != nil {
		return err
	}
	creator, err := ecrecover(header, c.signatures)
	if err != nil {
		return err
	}

	if _, ok := snap.Signers[creator]; !ok {
		if common.IndexOf(validators, creator) == -1 {
			return errUnauthorizedSigner
		}
	}

	for seen, recent := range snap.Recents {
		if len(validators) <= 1 {
			break
		}
		if recent == creator {
			// Signer is among RecentsRLP, only fail if the current block doesn't shift it out
			// There is only case that we don't allow signer to create two continuous blocks.
			if limit := uint64(2); seen > number-limit {
				// Only take into account the non-epoch blocks
				if number%c.config.Epoch != 0 {
					return errUnauthorizedSigner
				}
			}
		}
	}

	// Ensure that the difficulty corresponds to the turn-ness of the signer
	parent := chain.GetHeader(header.ParentHash, number-1)
	difficulty := c.calcDifficulty(creator, parent.Number.Uint64(), parent.Hash(), chain)
	if header.Difficulty.Int64() != difficulty.Int64() {
		return errInvalidDifficulty
	}

	// Enforce double validation
	if number > c.config.Epoch {
		attestor, err := c.Attestor(header)
		if err != nil {
			return err
		}

		checkpointHeader := GetCheckpointHeader(c.config, parent, chain)
		valAttPairs, _, err := c.backend.PosvGetCreatorAttestorPairs(c, c.chainConfig, header, checkpointHeader, logger)
		if err != nil {
			return err
		}
		assignedAttestor, ok := valAttPairs[creator]
		if !ok || attestor != assignedAttestor {
			return errInvalidBlockAttestor
		}
	}

	return nil
}

// Verify the current validators list at checkpoint block are comformed to the consensus rules.
// Error with be returned violations are found.
func (c *Posv) verifyValidators(chain consensus.ChainReader, header *types.Header, parents []*types.Header, logger log.Logger) error {
	number := header.Number.Uint64()
	snap, err := c.snapshot(chain, header.Number.Uint64()-1, header.ParentHash, parents)
	if err != nil {
		return err
	}

	validators := snap.signers()
	retryCount := 0
	for retryCount < 2 {
		// compare penalties computed from state with header.Penalties
		penalties, err := c.backend.PosvGetPenalties(c, c.chainConfig, c.chainConfig.Posv, c.chainConfig.Viction, header, chain, logger)
		if err != nil {
			return err
		}
		penaltiesBuff := EncodePenaltiesForHeader(penalties)
		if !bytes.Equal(penaltiesBuff, header.Penalties) {
			return errInvalidCheckpointPenalties
		}

		// remove penalized validators in current epoch
		if len(penalties) > 0 {
			validators = common.SetSubstract(validators, penalties)
			header.Penalties = EncodePenaltiesForHeader(penalties)
		}
		// remove penalized validators in recent epochs
		for i := uint64(1); i <= c.chainConfig.Viction.PenaltyEpochCount; i++ {
			prevCheckpointBlockNumber := number - (i * c.config.Epoch)
			prevCehckpointHeader := chain.GetHeaderByNumber(prevCheckpointBlockNumber)
			penalties := DecodePenaltiesFromHeader(prevCehckpointHeader.Penalties)
			if len(penalties) > 0 {
				validators = common.SetSubstract(validators, penalties)
			}
		}
		// compare validators computed from state with header.Extra
		headerValidators := ExtractValidatorsFromCheckpointHeader(header)
		if common.AreSimilarSlices(validators, headerValidators) {
			break
		}

		// if not matched, try to get validators from smart contract and verify again
		if retryCount == 0 {
			gapBlockNumber := number - c.config.Gap
			gapBlockHeader := chain.GetHeaderByNumber(gapBlockNumber)
			validators, err = c.backend.PosvGetValidators(c.chainConfig.Viction, gapBlockHeader, logger)
			if err != nil {
				return err
			}
		}
		// maximum retry reached, return error
		if retryCount == 1 {
			return errInvalidCheckpointValidators
		}

		retryCount++
	}

	return nil
}
