// Copyright 2016 The go-ethereum Authors
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

// Package posv implements the proof-of-stake-voting consensus engine.
package posv

import (
	"bytes"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	extraVanity = 32 // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal   = 65 // Fixed number of extra-data suffix bytes reserved for signer seal

	nonceAuthVote = hexutil.MustDecode("0xffffffffffffffff") // Magic nonce number to vote on adding a new signer
	nonceDropVote = hexutil.MustDecode("0x0000000000000000") // Magic nonce number to vote on removing a signer.

	uncleHash = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
)

// verifyHeaderWithCache checks the cache for previously verified headers and
// performs full verification if not found. Successfully verified headers are
// cached to avoid redundant checks.
func (c *Posv) verifyHeaderWithCache(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	_, check := c.verifiedBlocks.Get(header.Hash())
	if check {
		return nil
	}
	err := c.verifyHeader(chain, header, parents)
	if err == nil {
		c.verifiedBlocks.Add(header.Hash(), true)
	}
	return err
}

// verifyHeader checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (c *Posv) verifyHeader(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}
	number := header.Number.Uint64()

	now := time.Now()
	nowUnix := now.Unix()

	// Don't waste time checking blocks from the future
	if header.Time > uint64(nowUnix) {
		return consensus.ErrFutureBlock
	}

	// Checkpoint blocks need to enforce zero beneficiary
	checkpoint := (number % c.config.Epoch) == 0
	if checkpoint && header.Coinbase != (common.Address{}) {
		return errInvalidCheckpointBeneficiary
	}

	// Nonces must be 0x00..0 or 0xff..f, zeroes enforced on checkpoints
	if !bytes.Equal(header.Nonce[:], nonceAuthVote) && !bytes.Equal(header.Nonce[:], nonceDropVote) {
		return errInvalidVote
	}

	if checkpoint && !bytes.Equal(header.Nonce[:], nonceDropVote) {
		return errInvalidCheckpointVote
	}

	// Check that the extra-data contains both the vanity and signature
	if len(header.Extra) < extraVanity {
		return errMissingVanity
	}
	if len(header.Extra) < extraVanity+extraSeal {
		return errMissingSignature
	}
	// Ensure that the extra-data contains a signer list on checkpoint, but none otherwise
	signersBytes := len(header.Extra) - extraVanity - extraSeal
	if !checkpoint && signersBytes != 0 {
		return errExtraSigners
	}
	if checkpoint && signersBytes%common.AddressLength != 0 {
		return errInvalidCheckpointSigners
	}
	// Ensure that the mix digest is zero as we don't have fork protection currently
	if header.MixDigest != (common.Hash{}) {
		return errInvalidMixDigest
	}
	// Ensure that the block doesn't contain any uncles which are meaningless in PoA
	if header.UncleHash != uncleHash {
		return errInvalidUncleHash
	}

	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyForkHashes(chain.Config(), header, false); err != nil {
		return err
	}

	// All basic checks passed, verify cascading fields
	return c.verifyCascadingFields(chain, header, parents)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (c *Posv) verifyCascadingFields(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) error {
	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}

	// Retrieve the snapshot needed to verify this header and cache it
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}

	if parent.Time+c.config.Period > header.Time {
		return ErrInvalidTimestamp
	}

	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := c.snapshot(chain, number-1, header.ParentHash, parents)
	if err != nil {
		log.Debug("Failed to retrieve snapshot", "number", number, "err", err)
		return err
	}

	// If the block is a checkpoint block, verify the signer list
	if number%c.config.Epoch == 0 {
		chain := chain.(consensus.ChainReader)
		err := c.verifyValidators(chain, header, parents)
		if err != nil {
			log.Debug("Failed to verify validators", "number", number, "err", err)
			return err
		}
	}

	// All basic checks passed, verify the seal and return
	return c.verifySeal(chain, header, snap)

}

func (c *Posv) verifyValidators(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	number := header.Number.Uint64()
	snap, err := c.snapshot(chain, header.Number.Uint64()-1, header.ParentHash, parents)
	if err != nil {
		return err
	}

	validators := snap.GetSigners()
	retryCount := 0
	for retryCount < 2 {
		// compare penalties computed from state with header.Penalties
		penalties, err := c.backend.PosvGetPenalties(c, chain.Config(), c.config, chain.Config().Viction, header, chain)
		if err != nil {
			return err
		}

		penaltiesBuff := EncodePenaltiesForHeader(penalties)
		if !bytes.Equal(penaltiesBuff, header.Penalties) {
			return errInvalidCheckpointPenalties
		}
		// remove penalized validators in current epoch
		if len(penalties) > 0 {
			validators = common.RemoveItemFromArray(validators, penalties)
			header.Penalties = EncodePenaltiesForHeader(penalties)
		}
		// remove penalized validators in recent epochs
		for i := uint64(1); i <= chain.Config().Viction.PenaltyEpochCount; i++ {
			prevCheckpointBlockNumber := number - (i * c.config.Epoch)
			prevCehckpointHeader := chain.GetHeaderByNumber(prevCheckpointBlockNumber)
			penalties := DecodePenaltiesFromHeader(prevCehckpointHeader.Penalties)
			if len(penalties) > 0 {
				validators = common.RemoveItemFromArray(validators, penalties)
			}
		}
		// compare validators computed from state with header.Extra
		headerValidators := ExtractValidatorsFromCheckpointHeader(header)
		validValidators := common.AreSimilarSlices(headerValidators, validators)
		if validValidators {
			break
		}
		// if not matched, try to get validators from smart contract and verify again
		if retryCount == 0 {
			gapBlockNumber := number - c.config.Gap
			gapBlockHeader := chain.GetHeaderByNumber(gapBlockNumber)
			validators, err = c.backend.PosvGetValidators(chain.Config().Viction, gapBlockHeader, chain)
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

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements.
func (c *Posv) verifySeal(chainH consensus.ChainHeaderReader, header *types.Header, snap *Snapshot) error {
	chain := chainH.(consensus.ChainReader)
	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	// Resolve the authorization key and check against signers
	validators, err := c.backend.PosvGetValidators(chain.Config().Viction, header, chain)
	if err != nil {
		log.Debug("Failed to get validators", "number", number, "err", err)
		return err
	}
	creator, err := ecrecover(header, c.signatures)
	if err != nil {
		log.Debug("Failed to recover signer", "number", number, "err", err)
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
		valAttPairs, _, err := c.backend.PosvGetCreatorAttestorPairs(c, chain.Config(), header, checkpointHeader)
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

func (c *Posv) snapshot(chain consensus.ChainHeaderReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	return nil, nil
}
