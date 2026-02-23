package posv

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
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
	"github.com/ethereum/go-ethereum/trie"
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

	wiggleTime = 500 * time.Millisecond // Random delay (per signer) to allow concurrent signers
)

type Masternode struct {
	Address common.Address
	Stake   *big.Int
}

var (
	diffInTurn = big.NewInt(2) // Block difficulty for in-turn signatures
	diffNoTurn = big.NewInt(1) // Block difficulty for out-of-turn signatures
)

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
	errInvalidTimestamp = errors.New("invalid timestamp")

	// errUnauthorized is returned if a header is signed by a non-authorized entity.
	errUnauthorized = errors.New("unauthorized")

	// errInvalidDifficulty is returned if the difficulty of a block is not either
	// of 1 or 2, or if the value does not match the turn of the signer.
	errInvalidDifficulty = errors.New("invalid difficulty")

	errInvalidCheckpointValidators = errors.New("invalid validator list on checkpoint block")

	// ErrUnauthorizedSigner is returned if a header is signed by a non-authorized entity.
	errUnauthorizedSigner = errors.New("unauthorized signer")

	errInvalidBlockAttestor = errors.New("invalid block attestor")

	// errInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errInvalidVotingChain = errors.New("invalid voting chain")

	// that already signed a header recently, thus is temporarily not allowed to.
	errRecentlySigned = errors.New("recently signed")

	errEmptyValidators = errors.New("validators is empty")
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

// PosvRLP returns the rlp bytes which needs to be signed for the proof-of-authority
// sealing. The RLP to sign consists of the entire header apart from the 65 byte signature
// contained at the end of the extra data.
//
// Note, the method requires the extra data to be at least 65 bytes, otherwise it
// panics. This is done to avoid accidentally using both forms (signature present
// or not), which could be abused to produce different hashes for the same header.
func PosvRLP(header *types.Header) []byte {
	b := new(bytes.Buffer)
	encodeSigHeader(b, header)
	return b.Bytes()
}

// VerifyHeader checks whether a header conforms to the consensus rules.
func (c *Posv) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, _ bool) error {
	return c.verifyHeaderWithCache(chain, header, nil)
}

func (c *Posv) calcDifficulty(signer common.Address, parentNumber uint64, parentHash common.Hash, chain consensus.ChainHeaderReader) *big.Int {
	_, currentIndex, parentIndex, validatorCount, err := c.IsMyTurn(signer, parentNumber, parentHash, chain)
	if err == nil {
		distance := Distance(currentIndex, parentIndex, validatorCount)
		return big.NewInt(int64(validatorCount - distance + 1))
	}
	return big.NewInt(int64(validatorCount + currentIndex - parentIndex))

}

// Return the distance between current index and parent index in the circular list of validators.
func Distance(currentIndex, parentIndex, validatorCount int) int {
	if currentIndex > parentIndex {
		return currentIndex - parentIndex
	}
	return validatorCount + currentIndex - parentIndex
}

// Check if the signer is inturn to mint current block. Also return context of the check including:
// currentIndex, parentIndex, validatorCount.
func (c *Posv) IsMyTurn(signer common.Address, parentNumber uint64, parentHash common.Hash, chain consensus.ChainHeaderReader) (bool, int, int, int, error) {
	parent := chain.GetHeader(parentHash, parentNumber)
	checkpointHeader := GetCheckpointHeader(c.config, parent, chain)
	validators := ExtractValidatorsFromCheckpointHeader(checkpointHeader)
	validatorsCount := len(validators)
	if validatorsCount == 0 {
		return false, -1, -1, 0, errEmptyValidators
	}

	parentIndex := -1
	if parentNumber > 0 {
		parentCreator, err := c.Author(parent)
		if err != nil {
			return false, 0, 0, 0, err
		}
		parentIndex = common.IndexOf(validators, parentCreator)
	}
	currentIndex := common.IndexOf(validators, signer)

	inturn := (parentIndex+1)%validatorsCount == currentIndex
	return inturn, currentIndex, parentIndex, validatorsCount, nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the
// header for running the transactions on top.
func (c *Posv) Prepare(chainH consensus.ChainHeaderReader, header *types.Header) error {
	chain := chainH.(consensus.ChainReader)

	// If the block isn't a checkpoint, cast a random vote (good enough for now)
	header.Coinbase = common.Address{}
	header.Nonce = types.BlockNonce{}

	number := header.Number.Uint64()
	// Assemble the voting snapshot to check which votes make sense
	snap, err := c.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}
	c.lock.RLock()
	if number%c.config.Epoch != 0 {
		// Gather all the proposals that make sense voting on
		addresses := make([]common.Address, 0, len(c.proposals))
		for address, authorize := range c.proposals {
			if snap.validVote(address, authorize) {
				addresses = append(addresses, address)
			}
		}
		// If there's pending proposals, cast a vote on them
		if len(addresses) > 0 {
			header.Coinbase = addresses[rand.Intn(len(addresses))] // nolint: gosec
			if c.proposals[header.Coinbase] {
				copy(header.Nonce[:], nonceAuthVote)
			} else {
				copy(header.Nonce[:], nonceDropVote)
			}
		}
	}

	// Copy signer protected by mutex to avoid race condition
	signer := c.signer
	c.lock.RUnlock()

	// Set the correct difficulty
	header.Difficulty = c.calcDifficulty(signer, number-1, header.ParentHash, chain)

	// Ensure the extra data has all its components
	if len(header.Extra) < ExtraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, ExtraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:ExtraVanity]

	if number%c.config.Epoch == 0 {
		validators := snap.GetSigners()
		// remove penalized validators in current epoch
		penalties, err := c.backend.PosvGetPenalties(c, chain.Config(), c.config, chain.Config().Viction, header, chain)
		if err != nil {
			return err
		}
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
		// Write the final list of validators to Extra field
		for _, validator := range validators {
			header.Extra = append(header.Extra, validator[:]...)
		}
		// Write list of attestors to NewAttestors field
		attestors, err := c.backend.PosvGetAttestors(*chain.Config().Viction, header, validators)
		if err != nil {
			return err
		}
		header.NewAttestors = EncodeAttestorsForHeader(attestors)
	}
	header.Extra = append(header.Extra, make([]byte, ExtraSeal)...)

	// Mix digest is reserved for now, set to empty
	header.MixDigest = common.Hash{}

	// Ensure the timestamp has the correct delay
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Time = parent.Time + c.config.Period

	now := uint64(time.Now().Unix())
	if header.Time < now {
		header.Time = now
	}

	return nil
}

func (c *Posv) FinalizeAndAssemble(chainH consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	chain := chainH.(consensus.ChainReader)
	// set block reward
	number := header.Number.Uint64()
	rCheckpoint := chain.Config().Posv.Epoch

	if number%rCheckpoint == 0 {
		rewards, err := c.backend.PosvGetEpochReward(c, chain.Config(), c.config, chain.Config().Viction, header, chain, state, log.Root())
		if err != nil {
			return nil, err
		}
		if len(params.StoreRewardFolder) > 0 {
			data, err := json.Marshal(rewards)
			if err == nil {
				err = ioutil.WriteFile(filepath.Join(params.StoreRewardFolder, header.Number.String()+"."+header.Hash().Hex()), data, 0644)
			}
			if err != nil {
				log.Error("Error when save reward info ", "number", header.Number, "hash", header.Hash().Hex(), "err", err)
			}
		}
	}
	// the state remains as is and uncles are dropped
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)

	return types.NewBlock(header, txs, nil, receipts, new(trie.Trie)), nil
}

// Close implements consensus.Engine. It's a noop for clique as there are no background threads.
func (c *Posv) Close() error {
	return nil
}

// Finalize implements consensus.Engine, ensuring no uncles are set, nor block
// rewards given, and returns the final block.
func (c *Posv) Finalize(chainH consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	chain := chainH.(consensus.ChainReader)
	// set block reward
	number := header.Number.Uint64()
	rCheckpoint := chain.Config().Posv.Epoch
	//[TO-DO] Return nil if backend is not set, as this method is called by both miner and full node, and full node doesn't need to calculate reward. --- IGNORE ---
	if c.backend == nil {
		return
	}

	if number%rCheckpoint == 0 {
		rewards, err := c.backend.PosvGetEpochReward(c, chain.Config(), c.config, chain.Config().Viction, header, chain, state, log.Root())
		if err != nil {
			log.Error("Error when getting epoch reward ", "number", header.Number, "hash", header.Hash().Hex(), "err", err)
			return
		}
		if len(params.StoreRewardFolder) > 0 {
			data, err := json.Marshal(rewards)
			if err == nil {
				err = ioutil.WriteFile(filepath.Join(params.StoreRewardFolder, header.Number.String()+"."+header.Hash().Hex()), data, 0644)
			}
			if err != nil {
				log.Error("Error when save reward info ", "number", header.Number, "hash", header.Hash().Hex(), "err", err)
			}
		}
	}
	// the state remains as is and uncles are dropped
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)
}

// APIs implements consensus.Engine, returning the user facing RPC API to allow
// controlling the signer voting.
func (c *Posv) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return []rpc.API{{
		Namespace: "posv",
		Version:   "1.0",
		Service:   &API{chain: chain, posv: c},
		Public:    false,
	}}
}

// Seal implements consensus.Engine, attempting to create a sealed block using
// the local signing credentials.
func (c *Posv) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()

	// Sealing the genesis block is not supported
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	// For 0-period chains, refuse to seal empty blocks (no reward but would spin sealing)
	if c.config.Period == 0 && len(block.Transactions()) == 0 {
		log.Info("Sealing paused, waiting for transactions")
		results <- nil

		return nil
	}
	// Don't hold the signer fields for the entire sealing procedure
	c.lock.RLock()
	signer, signFn := c.signer, c.signFn
	c.lock.RUnlock()

	// Bail out if we're unauthorized to sign a block
	snap, err := c.snapshot(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}
	if _, authorized := snap.Signers[signer]; !authorized {
		return fmt.Errorf("Posv.Seal: %w", errUnauthorizedSigner)
	}
	// If we're amongst the recent signers, wait for the next block
	for seen, recent := range snap.Recents {
		if recent == signer {
			// Signer is among RecentsRLP, only wait if the current block doesn't shift it out
			if limit := uint64(len(snap.Signers)/2 + 1); number < limit || seen > number-limit {
				log.Info("Signed recently, must wait for others")
				return nil
			}
		}
	}
	// Sweet, the protocol permits us to sign the block, wait for our time
	delay := time.Unix(int64(header.Time), 0).Sub(time.Now()) // nolint: gosimple
	if header.Difficulty.Cmp(diffNoTurn) == 0 {
		// It's not our turn explicitly to sign, delay it a bit
		wiggle := time.Duration(len(snap.Signers)/2+1) * wiggleTime
		delay += time.Duration(rand.Int63n(int64(wiggle))) // nolint: gosec

		log.Trace("Out-of-turn signing requested", "wiggle", common.PrettyDuration(wiggle))
	}
	// Sign all the things!
	sighash, err := signFn(accounts.Account{Address: signer}, accounts.MimetypePosv, PosvRLP(header))
	if err != nil {
		return err
	}
	copy(header.Extra[len(header.Extra)-ExtraSeal:], sighash)
	// Wait until sealing is terminated or delay timeout.
	log.Trace("Waiting for slot to sign and propagate", "delay", common.PrettyDuration(delay))
	go func() {
		select {
		case <-stop:
			return
		case <-time.After(delay):
		}

		select {
		case results <- block.WithSeal(header):
		default:
			log.Warn("Sealing result is not read by miner", "sealhash", SealHash(header))
		}
	}()

	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have based on the previous blocks in the chain and the
// current signer.
func (c *Posv) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	return c.calcDifficulty(c.signer, parent.Number.Uint64(), parent.Hash(), chain)
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
	return c.verifySeal(chain, header, nil)
}

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
func (c *Posv) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := c.verifyHeaderWithCache(chain, header, headers[:i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// Author implements consensus.Engine, returning the Ethereum address recovered
// from the signature in the header's extra-data section.
// This is thread-safe (only access the header, as well as signatures, which
// are lru.ARCCache, which is thread-safe)
func (c *Posv) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header, c.signatures)
}

// Get signer coinbase
func (c *Posv) Signer() common.Address { return c.signer }
