package posv

import (
	"bytes"
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

	// chainWithCurrentBlock is satisfied by *core.BlockChain, whose CurrentBlock()
	// only advances after the full block (with state trie) has been committed.
	// *core.HeaderChain also satisfies the shape but returns nil, meaning no
	// full block / state is available in that path (downloader header pre-validation).
	type chainWithCurrentBlock interface {
		CurrentBlock() *types.Block
	}

	go func() {
		for i, header := range headers {
			number := header.Number.Uint64()
			// For checkpoint blocks, PosvGetPenalties / PosvGetValidators need to read
			// the chain state at the gap block (i.e. checkpointNumber - 1). We must wait
			// until that block and its state are fully committed to the DB before verifying
			// the checkpoint header.
			if c.config != nil && number > 0 && number%c.config.Epoch == 0 {
				requiredBlock := number - 1
				if cbc, ok := chain.(chainWithCurrentBlock); ok {
					lastLog := time.Now()
					for {
						select {
						case <-abort:
							return
						default:
						}
						cb := cbc.CurrentBlock()
						if cb == nil {
							// Header-only chain: state never committed here.
							// Skip the wait; verifyValidators handles missing state.
							break
						}
						if cb.NumberU64() >= requiredBlock {
							break
						}
						if time.Since(lastLog) >= 5*time.Second {
							log.Info("VerifyHeaders: waiting for gap block state before verifying checkpoint",
								"checkpoint", number, "requiredBlock", requiredBlock,
								"currentBlock", cb.NumberU64())
							lastLog = time.Now()
						}
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

// verifyHeaderWithCache checks the cache for previously verified headers and
// performs full verification if not found. Successfully verified headers are
// cached to avoid redundant checks.
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
func (c *Posv) verifyCascadingFields(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header, seal bool) error {
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
		return errInvalidTimestamp
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}

	// If the block is a checkpoint block, verify the signer list
	if number%c.config.Epoch == 0 {
		chain, ok := chain.(consensus.ChainReader)
		if !ok {
			log.Error("No chain reader provided for checkpoint verification")
			return fmt.Errorf("no chain reader provided for checkpoint verification")
		}
		err := c.verifyValidators(chain, header, parents)

		if err != nil {
			return err
		}
	}

	// All basic checks passed, verify the seal and return
	return c.verifySeal(chain, header, parents, seal)

}

func (c *Posv) verifyValidators(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	if c.backend == nil {
		return errBackendNotSet
	}
	number := header.Number.Uint64()
	log.Debug("Verifying checkpoint validators", "number", number, "hash", header.Hash().Hex())

	// ignore signerCheck at checkpoint block 14458500 due to wrong snapshot at gap 14458495
	if number == chain.Config().TIPFixSignerCheckBlock.Uint64() {
		return nil
	}

	// Load snapshot at the gap block (checkpoint - Gap), where UpdateMasternodes
	// stored the updated snapshot. Resolve the gap block hash from DB first,
	// then fall back to the in-batch parents slice.
	// Pass the parents slice to snapshot so it can walk backward through
	// in-batch blocks if the gap block snapshot is not yet in c.recents.
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
		// Fallback: try to get snapshot at parent block
		fallbackSnap, fallbackErr := c.snapshot(chain, number-1, header.ParentHash, parents)
		if fallbackErr != nil {
			// Both ways failed, return error
			return fallbackErr
		}
		// Use snapshot fallback, assign it to snap and continue processing below
		snap = fallbackSnap
	}
	headerValidators := ExtractValidatorsFromCheckpointHeader(header)

	// Remove penalties recorded on the last PenaltyEpochCount checkpoint headers
	subtractRecentPenalties := func(vs []common.Address) ([]common.Address, error) {
		for i := uint64(1); i <= chain.Config().Viction.PenaltyEpochCount; i++ {
			if number <= (i * c.config.Epoch) {
				continue
			}
			prevCheckpointBlockNumber := number - (i * c.config.Epoch)
			prevCheckpointHeader := chain.GetHeaderByNumber(prevCheckpointBlockNumber)
			if prevCheckpointHeader == nil {
				return nil, fmt.Errorf("couldn't retrieve previous checkpoint header for penalty verification")
			}
			prevPenalties := DecodePenaltiesFromHeader(prevCheckpointHeader.Penalties)
			if len(prevPenalties) > 0 {
				log.Debug("Removing recent epoch penalties", "number", number,
					"epochAgo", i, "checkpointNumber", prevCheckpointBlockNumber, "penalties", prevPenalties)
				vs = common.SetSubstract(vs, prevPenalties)
			}
		}
		return vs, nil
	}

	// Shared validator verifier: runs penalties, validator list, and attestors checks
	// against a given validator set. Returns nil on full match or a concrete error.
	validateWithValidators := func(baseValidators []common.Address) error {
		// ExtractValidatorsFromCheckpointHeader reads checkpoint masternodes from header.Extra
		penalties, err := c.backend.PosvGetPenalties(c, chain.Config(), c.config, chain.Config().Viction, header, chain, baseValidators)
		if err != nil {
			return err
		}

		penaltiesBuff := EncodePenaltiesForHeader(penalties)
		if !bytes.Equal(penaltiesBuff, header.Penalties) {
			log.Error("Penalty mismatch", "number", number,
				"computedPenalties", penalties, "headerPenalties", DecodePenaltiesFromHeader(header.Penalties))
			return errInvalidCheckpointPenalties
		}

		// signers with current-epoch penalties removed.
		workingValidators := baseValidators
		if len(penalties) > 0 {
			log.Info("Removing current epoch penalties", "number", number, "penalties", penalties)
			workingValidators = common.SetSubstract(workingValidators, penalties)
		}
		workingValidators, err = subtractRecentPenalties(workingValidators)
		if err != nil {
			return err
		}
		if !common.AreSimilarSlices(headerValidators, workingValidators) {
			log.Info("Checkpoint validator mismatch", "number", number, "computedValidators", workingValidators, "headerValidators", headerValidators)
			return errInvalidCheckpointValidators
		}

		attestors, aerr := c.backend.PosvGetAttestors(chain.Config().Viction, header, workingValidators)
		if aerr != nil {
			log.Error("Checkpoint attestors lookup failed", "number", number, "err", aerr)
			return aerr
		}
		if !bytes.Equal(EncodeAttestorsForHeader(attestors), header.NewAttestors) {
			log.Error("NewAttestors mismatch", "number", number,
				"computed", attestors, "header", DecodeAttestorsFromHeader(header.NewAttestors))
			return errInvalidCheckpointNewAttestors
		}
		return nil
	}

	// 1) First, validate using validators from the snapshot (gap block).
	snapshotValidators := snap.signers()
	if err := validateWithValidators(snapshotValidators); err == nil {
		return nil
	} else {
		log.Warn("Checkpoint validator verify failed with snapshot validators, will try contract validators",
			"number", number, "err", err)
	}

	// 2) Fallback: re-run the same logic using validators read from the staking contract
	// over the [number-Gap, number-1) window. If this also fails, bubble up that error.
	var fetchErr error
	var contractValidators []common.Address
	for gap := number - c.config.Gap; gap < number; gap++ {
		gapHeader := chain.GetHeaderByNumber(gap)
		if gapHeader == nil {
			continue
		}
		vs, err := c.backend.PosvGetValidators(chain.Config().Viction, gapHeader, chain)
		if err == nil && len(vs) > 0 {
			log.Info("Validators from smart contract", "checkpoint", number, "gapBlock", gap, "validators", vs)
			contractValidators = vs
			break
		}
		fetchErr = err
		log.Debug("PosvGetValidators failed or returned empty, trying next block",
			"checkpoint", number, "gapBlockNumber", gap, "err", err)
	}
	if len(contractValidators) == 0 {
		return fetchErr
	}
	return validateWithValidators(contractValidators)
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements.
func (c *Posv) verifySeal(chainH consensus.ChainHeaderReader, header *types.Header, parents []*types.Header, seal bool) error {
	chain, ok := chainH.(consensus.ChainReader)
	if !ok {
		log.Error("No chain reader provided for checkpoint verification")
		return fmt.Errorf("no chain reader provided for checkpoint verification")
	}

	// Verifying the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}

	// Current epoch checkpoint: used for authorization and attestor checks.
	checkpointHeader := GetCheckpointHeader(c.config, header, chain, parents)
	if checkpointHeader == nil {
		return fmt.Errorf("cannot get checkpoint header: %d", number)
	}
	validators := ExtractValidatorsFromCheckpointHeader(checkpointHeader)

	// Previous epoch checkpoint: used for difficulty calculation.
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	prevCheckpointHeader := GetCheckpointHeader(c.config, parent, chain, parents)
	if prevCheckpointHeader == nil {
		return fmt.Errorf("cannot get checkpoint header: %d", number)
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
		log.Debug("Failed to recover signer", "number", number, "err", err)
		return err
	}
	if _, ok := snap.Signers[creator]; !ok {
		if common.IndexOf(validators, creator) == -1 {
			return errUnauthorizedSigner
		}
	}

	// Validate recency: prevent a signer from sealing two consecutive blocks.
	for seen, recent := range snap.Recents {
		log.Trace("[7s62][POSV-verifier] recency check", "recent", recent, "creator", creator)
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

	// Validate difficulty
	if header.Difficulty.Int64() != c.calcDifficulty(creator, parent, prevValidators).Int64() {
		return errInvalidDifficulty
	}

	// Enforce double validation
	if number > c.config.Epoch && seal {
		attestor, err := c.Attestor(header)
		if err != nil {
			return err
		}
		valAttPairs, _, err := c.backend.PosvGetCreatorAttestorPairs(c, chain.Config(), header, checkpointHeader)
		if err != nil {
			return err
		}
		assignedAttestor, ok := valAttPairs[creator]
		if !ok || attestor != assignedAttestor {
			log.Info("Invalid attestor", "number", number, "creator", creator.Hex(), "attestor", attestor.Hex(), "assignedAttestor", assignedAttestor.Hex())
			return errInvalidBlockAttestor
		}
	}

	return nil
}
