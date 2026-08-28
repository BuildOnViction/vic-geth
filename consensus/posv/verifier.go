// Copyright 2017 The go-ethereum Authors
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

package posv

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *Posv) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	return c.verifyHeaderWithCache(chain, header, nil, seal)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (c *Posv) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	// There are two types statisfy this interface
	// - *core.BlockChain: CurrentBlock() return latest block, only advances after the full block (with state trie) has been committed.
	// - *core.HeaderChain: CurrentBlock() return nil, meaning no full block is available.
	type chainWithCurrentBlock interface {
		CurrentBlock() *types.Block
	}

	go func() {
		for i, header := range headers {
			number := header.Number.Uint64()
			// PoSV need state data from the gap block for checkpoint computation.
			// We must wait until that block and its state are fully committed to the DB before verifying checkpoint header.
			if c.config != nil && number > 0 && number%c.config.Epoch == 0 {
				requiredBlock := number - 1
				if cbc, ok := chain.(chainWithCurrentBlock); ok {
					retryCount := 0
					lastcheck := time.Now()
					for {
						select {
						case <-abort:
							return
						default:
						}
						if retryCount > 120 {
							break
						}
						cb := cbc.CurrentBlock()
						if cb == nil {
							break
						}
						if cb.NumberU64() >= requiredBlock {
							break
						}
						if time.Since(lastcheck) >= time.Duration(c.config.Period)*time.Second {
							log.Info("[PoSV][VerifyHeader] Waiting for gap block to be available",
								"checkpoint", number, "requiredBlock", requiredBlock, "currentBlock", cb.NumberU64(),
							)
							retryCount++
							lastcheck = time.Now()
						}
						time.Sleep(250 * time.Millisecond)
					}
				}
			}

			err := c.verifyHeaderWithCache(chain, header, headers[:i], seals[i])
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()

	return abort, results
}

// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (c *Posv) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (c *Posv) VerifySeal(chain consensus.ChainHeaderReader, header *types.Header) error {
	return c.verifySeal(chain, header, nil, true)
}

// Checks whether a header conforms to the consensus rules.
func (c *Posv) verifyHeaderWithCache(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header, seal bool) error {
	if header == nil {
		return errUnknownBlock
	}
	_, check := c.verifiedBlocks.Get(header.Hash())
	if check {
		return nil
	}
	err := c.verifyHeader(chain, header, parents, seal)
	if err == nil {
		c.verifiedBlocks.Add(header.Hash(), true)
	}
	return err
}

// verifyHeader checks whether a header conforms to the consensus rules.The
// caller may optionally pass in a batch of parents (ascending order) to avoid
// looking those up from the database. This is useful for concurrently verifying
// a batch of new headers.
func (c *Posv) verifyHeader(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header, seal bool) error {
	if header.Number == nil {
		return errUnknownBlock
	}
	number := header.Number.Uint64()
	now := time.Now()
	nowUnix := now.Unix()

	if seal {
		if header.Number.Uint64() > c.config.Epoch && len(header.Attestor) == 0 {
			return consensus.ErrNoValidatorSignature
		}
		// Don't waste time checking blocks from the future
		if header.Time > uint64(nowUnix) {
			return consensus.ErrFutureBlock
		}
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
	if len(header.Extra) < ExtraVanity {
		return errMissingVanity
	}
	if len(header.Extra) < ExtraVanity+ExtraSeal {
		return errMissingSignature
	}
	// Ensure that the extra-data contains a signer list on checkpoint, but none otherwise
	signersBytes := len(header.Extra) - ExtraVanity - ExtraSeal
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
	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyForkHashes(chain.Config(), header, false); err != nil {
		return err
	}
	// All basic checks passed, verify cascading fields
	return c.verifyCascadingFields(chain, header, parents, seal)
}

// verifyCascadingFields verifies all the header fields that are not standalone,
// rather depend on a batch of previous headers. The caller may optionally pass
// in a batch of parents (ascending order) to avoid looking those up from the
// database. This is useful for concurrently verifying a batch of new headers.
func (c *Posv) verifyCascadingFields(chainH consensus.ChainHeaderReader, header *types.Header, parents []*types.Header, seal bool) error {
	chain, ok := chainH.(consensus.ChainReader)
	if !ok {
		return errNoChainReader
	}

	// The genesis block is the always valid dead-end
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}
	// Ensure that the block's timestamp isn't too close to its parent
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
		return errInvalidTimestamp
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}
	if !chain.Config().IsLondon(header.Number) {
		// Verify BaseFee not present before EIP-1559 fork.
		if header.BaseFee != nil {
			return fmt.Errorf("invalid baseFee before fork: have %d, want <nil>", header.BaseFee)
		}
	} else if err := misc.VerifyEip1559Header(chain.Config(), parent, header); err != nil {
		// Verify the header's EIP-1559 attributes.
		return err
	}
	// If the block is a checkpoint block, verify the signer list
	if number%c.config.Epoch == 0 {
		if err := c.verifyValidators(chain, header, parents); err != nil {
			return err
		}
	}
	// All basic checks passed, verify the seal and return
	return c.verifySeal(chain, header, parents, seal)
}

func (c *Posv) verifyValidators(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	// When core.NewBlockChain is called, verifyHeader flow will be invoked before the backend is set.
	if c.backend == nil {
		return errNoBackend
	}

	number := header.Number.Uint64()
	config := chain.Config()
	posvConfig := config.Posv
	victionConfig := config.Viction

	// Fix SignerCheck issue on Mainnet
	if number == config.TIPSignerCheckBlock.Uint64() {
		return nil
	}

	// Check that validators in the header (remote) can be recomputed from local state.
	validateRemoteHeader := func(remoteHeader *types.Header, localValidators []common.Address) error {
		// Validate penalties
		penalties, err := c.backend.PosvGetPenalties(c, config, posvConfig, victionConfig, remoteHeader, chain, localValidators)
		if err != nil {
			log.Error("[PoSV] verify header: cannot get local penalties", "number", number)
			return err
		}
		penaltiesBuff := EncodePenaltiesForHeader(penalties)
		if !bytes.Equal(remoteHeader.Penalties, penaltiesBuff) {
			remotePenalties := DecodePenaltiesFromHeader(remoteHeader.Penalties)
			log.Error("[PoSV] verify header: penalties mismatch", "number", number, "local", penalties, "header", remotePenalties)
			return errInvalidCheckpointPenalties
		}

		// Validate validators
		validators := localValidators
		if len(penalties) > 0 {
			validators = common.SetSubstract(validators, penalties)
		}
		previousPenalties := GetPenaltiesFromHeaders(posvConfig, number, config.Viction.PenaltyEpochCount, chain)
		for _, penalties := range previousPenalties {
			if len(penalties) > 0 {
				validators = common.SetSubstract(validators, penalties)
			}
		}
		if err != nil {
			log.Error("[PoSV] verify header: cannot get local validators", "number", number)
			return err
		}
		remoteValidators := ExtractValidatorsFromCheckpointHeader(remoteHeader)
		if !common.AreSimilarSlices(remoteValidators, validators) {
			log.Error("[PoSV] verify header: validators mismatch", "number", number, "local", validators, "header", remoteValidators)
			return errInvalidCheckpointValidators
		}

		// Validate new attestors
		attestors, err := c.backend.PosvGetAttestors(victionConfig, remoteHeader, validators)
		if err != nil {
			log.Error("[PoSV] verify header: cannot get local new attestors", "number", number)
			return err
		}
		if !bytes.Equal(EncodeAttestorsForHeader(attestors), remoteHeader.NewAttestors) {
			remoteNewAttestors := DecodeAttestorsFromHeader(remoteHeader.NewAttestors)
			log.Error("[PoSV] verify header: new attestors mismatch", "number", number, "local", attestors, "header", remoteNewAttestors)
			return errInvalidCheckpointNewAttestors
		}

		// All passed
		return nil
	}

	// Verify remote header using snapshot
	gapBlockNumber := number - c.config.Gap
	var gapBlockHash common.Hash
	if gapHeader := chain.GetHeaderByNumber(gapBlockNumber); gapHeader != nil {
		gapBlockHash = gapHeader.Hash()
	} else {
		for _, p := range parents {
			if p.Number.Uint64() == gapBlockNumber {
				gapBlockHash = p.Hash()
				break
			}
		}
	}
	snap, err := c.snapshot(chain, gapBlockNumber, gapBlockHash, parents)
	if err != nil {
		parentSnap, err := c.snapshot(chain, number-1, header.ParentHash, parents)
		if err != nil {
			return err
		}
		snap = parentSnap
	}
	snapshotValidators := snap.signersNext()
	if err := validateRemoteHeader(header, snapshotValidators); err == nil {
		return nil
	} else {
		log.Warn("[PoSV] Verify header from snapshot failed. Retry with state.", "number", number, "err", err)
	}

	// Retry to verify remote header using state.
	var lastErr error
	var contractValidators []common.Address
	for gapBlockNumber = number - posvConfig.Gap; gapBlockNumber < number; gapBlockNumber++ {
		gapHeader := chain.GetHeaderByNumber(gapBlockNumber)
		if gapHeader == nil {
			continue
		}
		validators, err := c.backend.PosvGetValidators(config, victionConfig, gapHeader, chain)
		if err == nil && len(validators) > 0 {
			contractValidators = validators
			break
		}
		lastErr = err
	}
	if len(contractValidators) == 0 {
		return lastErr
	}
	return validateRemoteHeader(header, contractValidators)
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.
func (c *Posv) verifySeal(chainH consensus.ChainHeaderReader, header *types.Header, parents []*types.Header, seal bool) error {
	// When core.NewBlockChain is called, verifyHeader flow will be invoked before the backend is set.
	if c.backend == nil {
		return errNoBackend
	}

	chain, ok := chainH.(consensus.ChainReader)
	if !ok {
		return errNoChainReader
	}

	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	// Current epoch checkpoint: used for authorization and attestor checks.
	checkpointHeader := GetCheckpointHeader(c.config, header, parents, chain)
	if checkpointHeader == nil {
		log.Error("[PoSV] verify header: cannot get checkpoint header", "number", number)
		return errCannotGetCheckpointHeader
	}
	validators := ExtractValidatorsFromCheckpointHeader(checkpointHeader)

	// Previous epoch checkpoint: used for difficulty calculation.
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	prevCheckpointHeader := GetCheckpointHeader(c.config, parent, parents, chain)
	if prevCheckpointHeader == nil {
		log.Error("[PoSV] verify header: cannot get checkpoint header", "number", number)
		return errCannotGetCheckpointHeader
	}
	prevValidators := ExtractValidatorsFromCheckpointHeader(prevCheckpointHeader)

	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := c.snapshot(chain, number-1, header.ParentHash, parents)
	if err != nil {
		return err
	}

	// Validate creator
	creator, err := ecrecover(header, c.signatures)
	if err != nil {
		return err
	}
	if _, ok := snap.Signers[creator]; !ok {
		if common.IndexOf(validators, creator) == -1 {
			log.Error("[PoSV] verify header: unauthorized signer", "number", number, "hash", header.Hash(), "address", creator)
			return errUnauthorizedSigner
		}
	}

	// Validate recency: prevent a signer from sealing two consecutive blocks.
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
					log.Error("[PoSV] verify header: recently signed", "number", number, "hash", header.Hash(), "signer", creator)
					return errRecentlySigned
				}
			}
		}
	}

	// Validate difficulty
	if header.Difficulty.Int64() != c.calcDifficulty(creator, parent, prevValidators).Int64() {
		log.Error("[PoSV] verify header: invalid difficulty", "number", number, "current", header.Difficulty.Int64(), "expected", c.calcDifficulty(creator, parent, prevValidators).Int64())
		return errInvalidDifficulty
	}

	config := chain.Config()
	posvConfig := config.Posv
	victionConfig := config.Viction

	// Enforce double validation
	if number > c.config.Epoch && seal {
		attestor, err := c.Attestor(header)
		if err != nil {
			return err
		}
		valAttPairs, _, err := c.backend.PosvGetCreatorAttestorPairs(config, posvConfig, victionConfig, header, checkpointHeader)
		if err != nil {
			return err
		}
		assignedAttestor, ok := valAttPairs[creator]
		if !ok || attestor != assignedAttestor {
			log.Error("[PoSV] verify header: invalid attestor", "number", number, "creator", creator.Hex(), "attestor", attestor.Hex(), "assignedAttestor", assignedAttestor.Hex())
			return errInvalidAttestor
		}
	}

	// All passed
	return nil
}
