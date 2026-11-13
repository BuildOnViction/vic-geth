// Copyright 2015 The go-ethereum Authors
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

package miner

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestExtractSpecialTransactionsForPosv(t *testing.T) {
	signer := types.HomesteadSigner{}
	keyA, _ := crypto.GenerateKey()
	keyB, _ := crypto.GenerateKey()
	addrA := crypto.PubkeyToAddress(keyA.PublicKey)
	addrB := crypto.PubkeyToAddress(keyB.PublicKey)

	special89 := common.HexToAddress("0x0000000000000000000000000000000000000089")
	special90 := common.HexToAddress("0x0000000000000000000000000000000000000090")
	normalTo := common.HexToAddress("0x00000000000000000000000000000000000000aa")

	// Account A: first three should be extracted (up to last special at nonce 2).
	a0, _ := types.SignTx(types.NewTransaction(0, special89, big.NewInt(1), 21000, big.NewInt(10), nil), signer, keyA)
	a1, _ := types.SignTx(types.NewTransaction(1, normalTo, big.NewInt(1), 21000, big.NewInt(9), nil), signer, keyA)
	a2, _ := types.SignTx(types.NewTransaction(2, special90, big.NewInt(1), 21000, big.NewInt(8), nil), signer, keyA)
	a3, _ := types.SignTx(types.NewTransaction(3, normalTo, big.NewInt(1), 21000, big.NewInt(7), nil), signer, keyA)

	// Account B: even with special destination, should not be extracted (not in signers allowlist).
	b0, _ := types.SignTx(types.NewTransaction(0, special89, big.NewInt(1), 21000, big.NewInt(6), nil), signer, keyB)
	b1, _ := types.SignTx(types.NewTransaction(1, normalTo, big.NewInt(1), 21000, big.NewInt(5), nil), signer, keyB)

	groups := map[common.Address]types.Transactions{
		addrA: types.Transactions{a0, a1, a2, a3},
		addrB: types.Transactions{b0, b1},
	}
	signers := map[common.Address]struct{}{
		addrA: {},
	}

	specialTxs := ExtractSpecialTransactionsForPosv(signer, groups, signers)

	if len(specialTxs) != 3 {
		t.Fatalf("expected 3 extracted special-prefix txs, got %d", len(specialTxs))
	}
	if specialTxs[0].Hash() != a0.Hash() || specialTxs[1].Hash() != a1.Hash() || specialTxs[2].Hash() != a2.Hash() {
		t.Fatalf("unexpected extracted special tx order")
	}

	txset := types.NewTransactionsByPriceAndNonce(signer, groups)
	var remain types.Transactions
	for tx := txset.Peek(); tx != nil; tx = txset.Peek() {
		remain = append(remain, tx)
		txset.Shift()
	}
	if len(remain) != 3 {
		t.Fatalf("expected 3 remaining txs, got %d", len(remain))
	}
	for _, tx := range remain {
		if tx.Hash() == a0.Hash() || tx.Hash() == a1.Hash() || tx.Hash() == a2.Hash() {
			t.Fatalf("extracted tx should not remain in normal tx set")
		}
	}
}

func TestExtractSpecialTransactionsForPosvNilSigners(t *testing.T) {
	signer := types.HomesteadSigner{}
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)
	special89 := common.HexToAddress("0x0000000000000000000000000000000000000089")
	normalTo := common.HexToAddress("0x00000000000000000000000000000000000000aa")

	t0, _ := types.SignTx(types.NewTransaction(0, special89, big.NewInt(1), 21000, big.NewInt(2), nil), signer, key)
	t1, _ := types.SignTx(types.NewTransaction(1, normalTo, big.NewInt(1), 21000, big.NewInt(1), nil), signer, key)
	groups := map[common.Address]types.Transactions{
		addr: types.Transactions{t0, t1},
	}

	specialTxs := ExtractSpecialTransactionsForPosv(signer, groups, nil)
	if len(specialTxs) != 0 {
		t.Fatalf("expected no special txs when signers is nil, got %d", len(specialTxs))
	}

	txset := types.NewTransactionsByPriceAndNonce(signer, groups)
	var got types.Transactions
	for tx := txset.Peek(); tx != nil; tx = txset.Peek() {
		got = append(got, tx)
		txset.Shift()
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 txs in normal set, got %d", len(got))
	}
	if got[0].Nonce() != 0 || got[1].Nonce() != 1 {
		t.Fatalf("unexpected nonce order for remaining txs")
	}
}
