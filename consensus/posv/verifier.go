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
		return errInvalidTimestamp
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

	//[TO-DO] if backend is not set, skip validator verification. This is to avoid circular dependency between posv and viction. --- IGNORE ---
	if c.backend == nil {
		return nil
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
			validators = common.SetSubstract(validators, penalties)
			header.Penalties = EncodePenaltiesForHeader(penalties)
		}
		// remove penalized validators in recent epochs
		for i := uint64(1); i <= chain.Config().Viction.PenaltyEpochCount; i++ {
			prevCheckpointBlockNumber := number - (i * c.config.Epoch)
			prevCehckpointHeader := chain.GetHeaderByNumber(prevCheckpointBlockNumber)
			penalties := DecodePenaltiesFromHeader(prevCehckpointHeader.Penalties)
			if len(penalties) > 0 {
				validators = common.SetSubstract(validators, penalties)
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
	// [TO-DO] skip function if backend is not set. This is to avoid circular dependency between posv and viction. --- IGNORE ---
	if c.backend == nil {
		return nil
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
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
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
				log.Trace("Loaded voting snapshot from disk", "number", number, "hash", hash)
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
		log.Trace("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}
	return snap, err
}
