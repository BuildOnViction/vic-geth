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
	"errors"
	"io"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/clique"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	lru "github.com/hashicorp/golang-lru"
	"golang.org/x/crypto/sha3"
)

const (
	inmemorySnapshots      = 128 // Number of recent vote snapshots to keep in memory
	blockSignersCacheLimit = 9000
	epochLength            = uint64(900) // Default number of blocks after which to checkpoint and reset the pending votes
	M2ByteLength           = 4
	AddressLength          = uint64(20)             // Length of an address
	ExtraVanity            = 32                     // Fixed number of extra-data prefix bytes reserved for signer vanity
	ExtraSeal              = crypto.SignatureLength // Fixed number of extra-data suffix bytes reserved for signer seal
)

type Masternode struct {
	Address common.Address
	Stake   *big.Int
}

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errInvalidCheckpointBeneficiary is returned if a checkpoint/epoch transition
	// block has a beneficiary set to non-zeroes.
	errInvalidCheckpointBeneficiary = errors.New("beneficiary in checkpoint block non-zero")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte suffix signature missing")

	// errInvalidVote is returned if a nonce value is something else that the two
	// allowed constants of 0x00..0 or 0xff..f.
	errInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")

	// errInvalidCheckpointVote is returned if a checkpoint/epoch transition block
	// has a vote nonce set to non-zeroes.
	errInvalidCheckpointVote = errors.New("vote nonce in checkpoint block non-zero")

	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the signer vanity.
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")

	// errExtraSigners is returned if non-checkpoint block contain signer data in
	// their extra-data fields.
	errExtraSigners = errors.New("non-checkpoint block contains extra signer list")

	// errInvalidCheckpointSigners is returned if a checkpoint block contains an
	// invalid list of signers (i.e. non divisible by 20 bytes, or not the correct
	// ones).
	errInvalidCheckpointSigners = errors.New("invalid signer list on checkpoint block")

	errInvalidCheckpointPenalties = errors.New("invalid penalty list on checkpoint block")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")

	// ErrInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	ErrInvalidTimestamp = errors.New("invalid timestamp")

	// errUnauthorized is returned if a header is signed by a non-authorized entity.
	errUnauthorized = errors.New("unauthorized")

	// errInvalidDifficulty is returned if the difficulty of a block is not either
	// of 1 or 2, or if the value does not match the turn of the signer.
	errInvalidDifficulty = errors.New("invalid difficulty")

	errInvalidCheckpointValidators = errors.New("invalid validator list on checkpoint block")

	// ErrUnauthorizedSigner is returned if a header is signed by a non-authorized entity.
	errUnauthorizedSigner = errors.New("unauthorized signer")

	errInvalidBlockAttestor = errors.New("invalid block attestor")
)

// sigHash returns the hash which is used as input for the proof-of-stake-voting
// signing. It is the hash of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
func sigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()

	rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-crypto.SignatureLength], // Yes, this will panic if extra is too short
		header.MixDigest,
		header.Nonce,
	})
	hasher.Sum(hash[:0])
	return hash
}

// ecrecover extracts the Ethereum account address from a signed header.
func ecrecover(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
	// If the signature's already cached, return that
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}
	// Retrieve the signature from the header extra-data
	if len(header.Extra) < ExtraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Extra[len(header.Extra)-ExtraSeal:]

	// Recover the public key and the Ethereum address
	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}

// Posv is the proof-of-stake-voting consensus engine proposed to support the
// Ethereum testnet following the Ropsten attacks.
type Posv struct {
	config *params.PosvConfig // Consensus engine configuration parameters
	db     ethdb.Database     // Database to store and retrieve snapshot checkpoints

	recents          *lru.ARCCache           // Snapshots for recent block to speed up reorgs
	signatures       *lru.ARCCache           // Signatures of recent blocks to speed up mining
	attestSignatures *lru.ARCCache           // Signatures of recent blocks to speed up mining
	verifiedBlocks   *lru.ARCCache           // Status of recent blocks to speed up syncing
	proposals        map[common.Address]bool // Current list of proposals we are pushing

	signer common.Address  // Ethereum address of the signing key
	signFn clique.SignerFn // Signer function to authorize hashes with
	lock   sync.RWMutex    // Protects the signer fields

	BlockSigners *lru.Cache

	// Hook for posv
	backend PosvBackend
}

// New creates a PoSV proof-of-stake-voting consensus engine with the initial
// signers set to the ones provided by the user.
func New(config *params.PosvConfig, db ethdb.Database) *Posv {
	// Set any missing consensus parameters to their defaults
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = epochLength
	}
	// Allocate the snapshot caches and create the engine
	BlockSigners, _ := lru.New(blockSignersCacheLimit)
	recents, _ := lru.NewARC(inmemorySnapshots)
	signatures, _ := lru.NewARC(inmemorySnapshots)
	attestSignatures, _ := lru.NewARC(inmemorySnapshots)
	verifiedBlocks, _ := lru.NewARC(inmemorySnapshots)
	return &Posv{
		config:           &conf,
		db:               db,
		BlockSigners:     BlockSigners,
		recents:          recents,
		signatures:       signatures,
		verifiedBlocks:   verifiedBlocks,
		attestSignatures: attestSignatures,
		proposals:        make(map[common.Address]bool),
	}
}

// Set the backend instance into PoSV for handling some features that require accessing to chain state.
// Must be called right after creation of PoSV.
func (c *Posv) SetBackend(backend PosvBackend) {
	c.backend = backend
}

func (c *Posv) Attestor(header *types.Header) (common.Address, error) {
	return ecrecover(header, c.attestSignatures)
}

// SealHash returns the hash of a block prior to it being sealed.
func (c *Posv) SealHash(header *types.Header) common.Hash {
	return SealHash(header)
}

// SealHash returns the hash of a block prior to it being sealed.
func SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()
	encodeSigHeader(hasher, header)
	hasher.Sum(hash[:0])
	return hash
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *Posv) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, _ bool) error {
	log.Info("VerifyHeader called", "number", header.Number.Uint64(), "hash", header.Hash().Hex(), "parentHash", header.ParentHash.Hex())
	err := c.verifyHeaderWithCache(chain, header, nil)
	if err != nil {
		log.Warn("VerifyHeader failed", "number", header.Number.Uint64(), "hash", header.Hash().Hex(), "err", err)
	} else {
		log.Info("VerifyHeader succeeded", "number", header.Number.Uint64(), "hash", header.Hash().Hex())
	}
	return err
}

// [TO-DO]
func (c *Posv) calcDifficulty(signer common.Address, parentNumber uint64, parentHash common.Hash, chain consensus.ChainHeaderReader) *big.Int {
	return nil
}

// [TO-DO]
// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *Posv) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	return nil
}

// [TO-DO]
func (c *Posv) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	return nil, nil
}

// [TO-DO]
// Close implements consensus.Engine. It's a noop for clique as there are no background threads.
func (c *Posv) Close() error {
	return nil
}

// [TO-DO]
// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (c *Posv) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	return
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
func (c *Posv) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header, c.signatures)
}

// [TO-DO]
// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *Posv) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{}
}

// [TO-DO]
// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (c *Posv) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	return errors.New("not implemented")
}

// [TO-DO]
// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (c *Posv) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return nil
}

// [TO-DO]]
// VerifyUncles implements consensus.Engine, always returning an error for any
// uncles as this consensus mechanism doesn't permit uncles.
func (c *Posv) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	return nil
}

// [TO-DO]
// VerifySeal implements consensus.Engine, checking whether the signature contained
// in the header satisfies the consensus protocol requirements.
func (c *Posv) VerifySeal(chain consensus.ChainHeaderReader, header *types.Header) error {
	return nil
}

// [TO-DO]
// encodeSigHeader encodes the header fields relevant for signing.
func encodeSigHeader(w io.Writer, header *types.Header) {
	enc := []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-crypto.SignatureLength], // Yes, this will panic if extra is too short
		header.MixDigest,
		header.Nonce,
	}
	if err := rlp.Encode(w, enc); err != nil {
		panic("can't encode: " + err.Error())
	}
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers. The
// method returns a quit channel to abort the operations and a results channel to
// retrieve the async verifications (the order is that of the input slice).
func (c *Posv) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, fullVerifies []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			// Determine if we should verify the seal for this header
			verifySeal := false
			if i < len(seals) {
				verifySeal = seals[i]
			}
			err := c.VerifyHeader(chain, header, verifySeal)

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// [TO-DO]
// Get signer coinbase
func (c *Posv) Signer() common.Address { return common.Address{} }
