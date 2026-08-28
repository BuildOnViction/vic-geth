// Copyright 2016 The go-ethereum Authors
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

package types

import (
	"container/heap"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strconv"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

var (
	// ErrInvalidOrderSig invalidate signer
	ErrInvalidOrderSig = errors.New("invalid transaction v, r, s values")
	errNoSignerOrder   = errors.New("missing signing methods")
)

const (
	OrderStatusNew           = "NEW"
	OrderStatusOpen          = "OPEN"
	OrderStatusPartialFilled = "PARTIAL_FILLED"
	OrderStatusFilled        = "FILLED"
	OrderStatusCancelled     = "CANCELLED"
	OrderTypeMo              = "MO"
	OrderTypeLo              = "LO"
)

// OrderTransaction order transaction
type OrderTransaction struct {
	data ordertxdata

	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

type ordertxdata struct {
	AccountNonce    uint64         `json:"nonce"    gencodec:"required"`
	Quantity        *big.Int       `json:"quantity,omitempty"`
	Price           *big.Int       `json:"price,omitempty"`
	ExchangeAddress common.Address `json:"exchangeAddress,omitempty"`
	UserAddress     common.Address `json:"userAddress,omitempty"`
	BaseToken       common.Address `json:"baseToken,omitempty"`
	QuoteToken      common.Address `json:"quoteToken,omitempty"`
	Status          string         `json:"status,omitempty"`
	Side            string         `json:"side,omitempty"`
	Type            string         `json:"type,omitempty"`
	OrderID         uint64         `json:"orderid,omitempty"`

	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	Hash common.Hash `json:"hash"`
}

// Check if the order is cancelled.
func (tx *OrderTransaction) IsCancelledOrder() bool {
	if tx.Status() == OrderStatusCancelled {
		return true
	}
	return false
}

// Check if the order is a Limit Order type.
func (tx *OrderTransaction) IsLoTypeOrder() bool {
	if tx.Type() == OrderTypeLo {
		return true
	}
	return false
}

// Check if the order is a Market Order type.
func (tx *OrderTransaction) IsMoTypeOrder() bool {
	if tx.Type() == OrderTypeMo {
		return true
	}
	return false
}

// EncodeRLP implements rlp.Encoder
func (tx *OrderTransaction) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx.data)
}

// DecodeRLP implements rlp.Decoder
func (tx *OrderTransaction) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(&tx.data)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}

	return err
}

func (tx *OrderTransaction) Nonce() uint64                   { return tx.data.AccountNonce }
func (tx *OrderTransaction) Quantity() *big.Int              { return tx.data.Quantity }
func (tx *OrderTransaction) Price() *big.Int                 { return tx.data.Price }
func (tx *OrderTransaction) ExchangeAddress() common.Address { return tx.data.ExchangeAddress }
func (tx *OrderTransaction) UserAddress() common.Address     { return tx.data.UserAddress }
func (tx *OrderTransaction) BaseToken() common.Address       { return tx.data.BaseToken }
func (tx *OrderTransaction) QuoteToken() common.Address      { return tx.data.QuoteToken }
func (tx *OrderTransaction) Status() string                  { return tx.data.Status }
func (tx *OrderTransaction) Side() string                    { return tx.data.Side }
func (tx *OrderTransaction) Type() string                    { return tx.data.Type }
func (tx *OrderTransaction) Signature() (V, R, S *big.Int)   { return tx.data.V, tx.data.R, tx.data.S }
func (tx *OrderTransaction) OrderHash() common.Hash          { return tx.data.Hash }
func (tx *OrderTransaction) OrderID() uint64                 { return tx.data.OrderID }
func (tx *OrderTransaction) EncodedSide() *big.Int {
	if tx.Side() == "BUY" {
		return big.NewInt(0)
	} else {
		return big.NewInt(1)
	}
}
func (tx *OrderTransaction) SetOrderHash(h common.Hash) { tx.data.Hash = h }

// From get transaction from
func (tx *OrderTransaction) From() *common.Address {
	if tx.data.V != nil {
		signer := OrderTxSigner{}
		if f, err := OrderSender(signer, tx); err != nil {
			return nil
		} else {
			return &f
		}
	} else {
		return nil
	}
}

// WithSignature returns a new transaction with the given signature.
// This signature needs to be formatted as described in the yellow paper (v+27).
func (tx *OrderTransaction) WithSignature(signer OrderSigner, sig []byte) (*OrderTransaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &OrderTransaction{data: tx.data}
	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

// ImportSignature make order tx with specific signature
func (tx *OrderTransaction) ImportSignature(V, R, S *big.Int) *OrderTransaction {

	if V != nil {
		tx.data.V = V
	}
	if R != nil {
		tx.data.R = R
	}
	if S != nil {
		tx.data.S = S
	}
	return tx
}

// Hash hashes the RLP encoding of tx.
// It uniquely identifies the transaction.
func (tx *OrderTransaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx)
	tx.hash.Store(v)
	return v
}

// CacheHash cache hash
func (tx *OrderTransaction) CacheHash() {
	v := rlpHash(tx)
	tx.hash.Store(v)
}

// Size returns the true RLP encoded storage size of the transaction, either by
// encoding and returning it, or returning a previsouly cached value.
func (tx *OrderTransaction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	c := writeCounter(0)
	rlp.Encode(&c, &tx.data)
	tx.size.Store(common.StorageSize(c))
	return common.StorageSize(c)
}

// NewOrderTransaction init order from value
func NewOrderTransaction(nonce uint64, quantity, price *big.Int, ex, ua, b, q common.Address, status, side, t string, hash common.Hash, id uint64) *OrderTransaction {
	return newOrderTransaction(nonce, quantity, price, ex, ua, b, q, status, side, t, hash, id)
}

func newOrderTransaction(nonce uint64, quantity, price *big.Int, ex, ua, b, q common.Address, status, side, t string, hash common.Hash, id uint64) *OrderTransaction {
	d := ordertxdata{
		AccountNonce:    nonce,
		Quantity:        new(big.Int),
		Price:           new(big.Int),
		ExchangeAddress: ex,
		UserAddress:     ua,
		BaseToken:       b,
		QuoteToken:      q,
		Status:          status,
		Side:            side,
		Type:            t,
		Hash:            hash,
		OrderID:         id,
		V:               new(big.Int),
		R:               new(big.Int),
		S:               new(big.Int),
	}
	if quantity != nil {
		d.Quantity.Set(quantity)
	}
	if price != nil {
		d.Price.Set(price)
	}

	return &OrderTransaction{data: d}
}

// OrderTransactions is a Transaction slice type for basic sorting.
type OrderTransactions []*OrderTransaction

// Len returns the length of s.
func (s OrderTransactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s.
func (s OrderTransactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp.
func (s OrderTransactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

// OrderTxDifference returns a new set t which is the difference between a to b.
func OrderTxDifference(a, b OrderTransactions) (keep OrderTransactions) {
	keep = make(OrderTransactions, 0, len(a))

	remove := make(map[common.Hash]struct{})
	for _, tx := range b {
		remove[tx.Hash()] = struct{}{}
	}

	for _, tx := range a {
		if _, ok := remove[tx.Hash()]; !ok {
			keep = append(keep, tx)
		}
	}

	return keep
}

// OrderTxByNonce sorted order by nonce defined
type OrderTxByNonce OrderTransactions

func (s OrderTxByNonce) Len() int           { return len(s) }
func (s OrderTxByNonce) Less(i, j int) bool { return s[i].data.AccountNonce < s[j].data.AccountNonce }

func (s OrderTxByNonce) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s *OrderTxByNonce) Push(x interface{}) {
	*s = append(*s, x.(*OrderTransaction))
}

func (s *OrderTxByNonce) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

type OrderTransactionByNonce struct {
	txs    map[common.Address]OrderTransactions
	heads  OrderTxByNonce
	signer OrderSigner
}

func NewOrderTransactionByNonce(signer OrderSigner, txs map[common.Address]OrderTransactions) *OrderTransactionByNonce {
	// Initialize a price based heap with the head transactions
	heads := make(OrderTxByNonce, 0, len(txs))
	for from, accTxs := range txs {
		heads = append(heads, accTxs[0])
		// Ensure the sender address is from the signer
		acc, _ := OrderSender(signer, accTxs[0])
		txs[acc] = accTxs[1:]
		if from != acc {
			delete(txs, from)
		}
	}
	heap.Init(&heads)

	// Assemble and return the transaction set
	return &OrderTransactionByNonce{
		txs:    txs,
		heads:  heads,
		signer: signer,
	}
}

// Peek returns the next transaction by price.
func (t *OrderTransactionByNonce) Peek() *OrderTransaction {
	if len(t.heads) == 0 {
		return nil
	}
	return t.heads[0]
}

// Shift replaces the current best head with the next one from the same account.
func (t *OrderTransactionByNonce) Shift() {
	acc, _ := OrderSender(t.signer, t.heads[0])
	if txs, ok := t.txs[acc]; ok && len(txs) > 0 {
		t.heads[0], t.txs[acc] = txs[0], txs[1:]
		heap.Fix(&t.heads, 0)
	} else {
		heap.Pop(&t.heads)
	}
}

// Pop removes the best transaction, *not* replacing it with the next one from
// the same account. This should be used when a transaction cannot be executed
// and hence all subsequent ones should be discarded from the same account.
func (t *OrderTransactionByNonce) Pop() {
	heap.Pop(&t.heads)
}

// OrderSigner interface for order transaction
type OrderSigner interface {
	// Equal returns true if the given signer is the same as the receiver.
	Equal(OrderSigner) bool

	// Hash returns the hash to be signed.
	Hash(tx *OrderTransaction) common.Hash

	// Sender returns the sender address of the transaction.
	Sender(tx *OrderTransaction) (common.Address, error)

	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx *OrderTransaction, sig []byte) (r, s, v *big.Int, err error)
}

type ordersigCache struct {
	signer OrderSigner
	from   common.Address
}

// OrderSender returns the address derived from the signature (V, R, S) using secp256k1
// elliptic curve and an error if it failed deriving or upon an incorrect
// signature.
//
// Sender may cache the address, allowing it to be used regardless of
// signing method. The cache is invalidated if the cached signer does
// not match the signer used in the current call.
func OrderSender(signer OrderSigner, tx *OrderTransaction) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(ordersigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}

	addr, err := signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	tx.from.Store(ordersigCache{signer: signer, from: addr})
	return addr, nil
}

// OrderSignTx signs the order transaction using the given order signer and private key
func OrderSignTx(tx *OrderTransaction, s OrderSigner, prv *ecdsa.PrivateKey) (*OrderTransaction, error) {
	h := s.Hash(tx)
	message := crypto.Keccak256(
		[]byte("\x19Ethereum Signed Message:\n32"),
		h.Bytes(),
	)

	sig, err := crypto.Sign(message[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}

// OrderTxSigner signer
type OrderTxSigner struct{}

// Equal compare two signer
func (ordersign OrderTxSigner) Equal(s2 OrderSigner) bool {
	_, ok := s2.(OrderTxSigner)
	return ok
}

// SignatureValues returns signature values. This signature needs to be in the [R || S || V] format where V is 0 or 1.
func (ordersign OrderTxSigner) SignatureValues(tx *OrderTransaction, sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}

// OrderCreateHash hash of new order
func (ordersign OrderTxSigner) OrderCreateHash(tx *OrderTransaction) common.Hash {
	sha := sha3.NewLegacyKeccak256()
	sha.Write(tx.ExchangeAddress().Bytes())
	sha.Write(tx.UserAddress().Bytes())
	sha.Write(tx.BaseToken().Bytes())
	sha.Write(tx.QuoteToken().Bytes())
	sha.Write(common.BigToHash(tx.Quantity()).Bytes())
	if tx.IsLoTypeOrder() {
		if tx.Price() != nil {
			sha.Write(common.BigToHash(tx.Price()).Bytes())
		}
	}
	sha.Write(common.BigToHash(tx.EncodedSide()).Bytes())
	sha.Write([]byte(tx.Status()))
	sha.Write([]byte(tx.Type()))
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Nonce()))).Bytes())
	return common.BytesToHash(sha.Sum(nil))
}

// OrderCancelHash hash of cancelled order
func (ordersign OrderTxSigner) OrderCancelHash(tx *OrderTransaction) common.Hash {
	sha := sha3.NewLegacyKeccak256()
	sha.Write(tx.OrderHash().Bytes())
	sha.Write(common.BigToHash(big.NewInt(int64(tx.Nonce()))).Bytes())
	sha.Write(tx.UserAddress().Bytes())
	sha.Write(common.BigToHash(big.NewInt(int64(tx.OrderID()))).Bytes())
	sha.Write([]byte(tx.Status()))
	sha.Write(tx.ExchangeAddress().Bytes())
	sha.Write(tx.BaseToken().Bytes())
	sha.Write(tx.QuoteToken().Bytes())

	return common.BytesToHash(sha.Sum(nil))
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (ordersign OrderTxSigner) Hash(tx *OrderTransaction) common.Hash {
	if tx.IsCancelledOrder() {
		return ordersign.OrderCancelHash(tx)
	}
	return ordersign.OrderCreateHash(tx)
}

// MarshalSignature encode signature
func MarshalSignature(R, S, V *big.Int) ([]byte, error) {
	sigBytes1 := common.BigToHash(R).Bytes()
	sigBytes2 := common.BigToHash(S).Bytes()
	v := big.NewInt(0)
	v.Sub(V, big.NewInt(27))

	bigstr := v.String()
	n, err := strconv.ParseInt(bigstr, 10, 8)
	if err != nil {
		return nil, err
	}
	sigBytes3 := byte(n)
	sigBytes := append([]byte{}, sigBytes1...)
	sigBytes = append(sigBytes, sigBytes2...)
	sigBytes = append(sigBytes, sigBytes3)

	return sigBytes, nil
}

// Sender get signer from
func (ordersign OrderTxSigner) Sender(tx *OrderTransaction) (common.Address, error) {

	message := crypto.Keccak256(
		[]byte("\x19Ethereum Signed Message:\n32"),
		ordersign.Hash(tx).Bytes(),
	)
	V, R, S := tx.Signature()

	sigBytes, err := MarshalSignature(R, S, V)
	if err != nil {
		return common.Address{}, err
	}
	pubKey, err := crypto.SigToPub(message, sigBytes)
	if err != nil {
		return common.Address{}, err
	}
	address := crypto.PubkeyToAddress(*pubKey)
	return address, nil

}

// CacheOrderSigner cache signed order
func CacheOrderSigner(signer OrderSigner, tx *OrderTransaction) {
	if tx == nil {
		return
	}
	addr, err := signer.Sender(tx)
	if err != nil {
		return
	}
	tx.from.Store(ordersigCache{signer: signer, from: addr})
}
