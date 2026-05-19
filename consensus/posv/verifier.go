package posv

import (
	"bytes"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	nonceAuthVote = hexutil.MustDecode("0xffffffffffffffff") // Magic nonce number to vote on adding a new signer
	nonceDropVote = hexutil.MustDecode("0x0000000000000000") // Magic nonce number to vote on removing a signer.

	uncleHash = types.CalcUncleHash(nil) // Always Keccak256(RLP([])) as uncles are meaningless outside of PoW.
)

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
		log.Error("[POSV] verifyHeader: Extra too short for seal",
			"number", header.Number, "extraLen", len(header.Extra),
			"extra", hexutil.Encode(header.Extra),
			"attestorLen", len(header.Attestor),
			"newAttestorsLen", len(header.NewAttestors),
			"posv", header.Posv,
			"hash", header.Hash().Hex())
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
	parent = resolveParent(chain, header, parents)
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}

	if parent.Time+c.config.Period > header.Time {
		return errInvalidTimestamp
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

// resolveParent returns the immediate parent of header, preferring the
// in-batch parents slice over a DB lookup.
func resolveParent(chain consensus.ChainHeaderReader, header *types.Header, parents []*types.Header) *types.Header {
	if len(parents) > 0 {
		return parents[len(parents)-1]
	}
	return chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
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
			return errInvalidNewAttestors
		}
		return nil
	}

	// 1) First, validate using validators from the snapshot (gap block).
	snapshotValidators := snap.GetSigners()
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
	if c.backend == nil {
		return errBackendNotSet
	}

	// Resolve the block immediately before header: prefer in-batch slice, fall back to DB.
	prevHeader := resolveParent(chain, header, parents)

	// Recover the block creator from the header seal.
	creator, err := ecrecover(header, c.signatures)
	if err != nil {
		log.Debug("Failed to recover signer", "number", number, "err", err)
		return err
	}

	// Checkpoint for the current epoch: used for authorization and attestor checks.
	checkpointHeader := GetCheckpointHeader(c.config, header, chain, parents)
	if checkpointHeader == nil {
		return fmt.Errorf("couldn't find checkpoint header for block %d", number)
	}
	validators := ExtractValidatorsFromCheckpointHeader(checkpointHeader)

	// Checkpoint for the previous block's epoch: used for difficulty calculation.
	// At an epoch boundary prevHeader belongs to the prior epoch, so its checkpoint
	// differs from the current one.
	prevCheckpointHeader := GetCheckpointHeader(c.config, prevHeader, chain, parents)
	if prevCheckpointHeader == nil {
		return fmt.Errorf("couldn't find checkpoint header for parent of block %d", number)
	}
	prevValidators := ExtractValidatorsFromCheckpointHeader(prevCheckpointHeader)

	if header.Difficulty.Int64() != c.calcDifficulty(creator, prevHeader, prevValidators).Int64() {
		return errInvalidDifficulty
	}

	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := c.snapshot(chain, number-1, header.ParentHash, parents)
	if err != nil {
		return err
	}

	if _, ok := snap.Signers[creator]; !ok {
		if common.IndexOf(validators, creator) == -1 {
			return errUnauthorizedSigner
		}
	}

	// [7s62] recency check: prevent a signer from sealing two consecutive blocks.
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

func (c *Posv) snapshot(chain consensus.ChainHeaderReader, number uint64, hash common.Hash, parents []*types.Header) (*Snapshot, error) {
	// Search for a snapshot in memory or on disk for checkpoints
	var (
		headers []*types.Header
		snap    *Snapshot
	)

	for snap == nil { //nolint:govet
		// If an in-memory snapshot was found, use that
		if s, ok := c.recents.Get(hash); ok {
			snap = s.(*Snapshot)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if (number+c.config.Gap)%c.config.Epoch == 0 {
			if s, err := loadSnapshot(c.config, c.signatures, c.db, hash); err == nil {
				snap = s
				break
			}
		}
		// If we're at the genesis, snapshot the initial state. Alternatively if we're
		// at a checkpoint block without a parent (light client CHT), or we have piled
		// up more headers than allowed to be reorged (chain reinit from a freezer),
		// consider the checkpoint trusted and snapshot it.
		if number == 0 || (number%c.config.Epoch == 0 && (len(headers) > params.FullImmutabilityThreshold || chain.GetHeaderByNumber(number-1) == nil)) {
			checkpoint := chain.GetHeaderByNumber(number)
			if checkpoint != nil {
				hash := checkpoint.Hash()

				signers := make([]common.Address, (len(checkpoint.Extra)-ExtraVanity-ExtraSeal)/common.AddressLength)
				for i := 0; i < len(signers); i++ {
					copy(signers[i][:], checkpoint.Extra[ExtraVanity+i*common.AddressLength:])
				}
				snap = newSnapshot(c.config, c.signatures, number, hash, signers)
				if err := snap.store(c.db); err != nil {
					return nil, err
				}
				log.Info("[PoSV] Stored checkpoint snapshot to disk", "number", number, "hash", hash)
				break
			}
		}
		// No snapshot for this header, gather the header and move backward
		var header *types.Header
		if len(parents) > 0 {
			// If we have explicit parents, pick from there (enforced)
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else {
			// No explicit parents (or no more left), reach out to the database
			header = chain.GetHeader(hash, number)
			if header == nil {
				return nil, consensus.ErrUnknownAncestor
			}
		}
		headers = append(headers, header)
		number, hash = number-1, header.ParentHash
	}
	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	c.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if (snap.Number+c.config.Gap)%c.config.Epoch == 0 && len(headers) > 0 {
		if err = snap.store(c.db); err != nil {
			return nil, err
		}
	}
	return snap, err
}
