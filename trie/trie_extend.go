// Copyright 2014 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"fmt"
)

// TryGetBestLeftKeyAndValue returns the key and value of the leftmost
// leaf in the trie. The returned key is in keybyte encoding, matching
// the keys passed to TryUpdate.
// The value bytes must not be modified by the caller.
// If the trie is empty, both return values are nil.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryGetBestLeftKeyAndValue() ([]byte, []byte, error) {
	key, value, newroot, didResolve, err := t.tryGetBestLeftKeyAndValue(t.root, []byte{})
	if err == nil && didResolve {
		t.root = newroot
	}
	if err != nil {
		return nil, nil, err
	}
	if len(key) == 0 {
		return nil, nil, nil
	}
	return hexToKeybytes(key), value, nil
}

// tryGetBestLeftKeyAndValue returns the leftmost leaf of the subtrie below
// origNode, carrying the hex-encoded key path accumulated in prefix. It
// resolves hash nodes on the way down and links the loaded children back
// into the trie on the way up.
func (t *Trie) tryGetBestLeftKeyAndValue(origNode node, prefix []byte) (key []byte, value []byte, newnode node, didResolve bool, err error) {
	switch n := (origNode).(type) {
	case nil:
		return nil, nil, nil, false, nil
	case *shortNode:
		switch v := n.Val.(type) {
		case valueNode:
			return append(prefix, n.Key...), v, n, false, nil
		default:
		}
		key, value, newnode, didResolve, err = t.tryGetBestLeftKeyAndValue(n.Val, append(prefix, n.Key...))
		if err == nil && didResolve {
			n = n.copy()
			n.Val = newnode
		}
		return key, value, n, didResolve, err
	case *fullNode:
		for i := 0; i < len(n.Children); i++ {
			if n.Children[i] == nil {
				continue
			}
			key, value, newnode, didResolve, err = t.tryGetBestLeftKeyAndValue(n.Children[i], append(prefix, byte(i)))
			if err == nil && didResolve {
				n = n.copy()
				n.Children[i] = newnode
			}
			return key, value, n, didResolve, err
		}
		return nil, nil, n, false, nil
	case hashNode:
		child, err := t.resolveHash(n, nil)
		if err != nil {
			return nil, nil, n, true, err
		}
		key, value, newnode, _, err := t.tryGetBestLeftKeyAndValue(child, prefix)
		return key, value, newnode, true, err
	default:
		return nil, nil, nil, false, fmt.Errorf("%T: invalid node: %v", origNode, origNode)
	}
}

// TryGetBestRightKeyAndValue returns the key and value of the rightmost
// leaf in the trie. The returned key is in keybyte encoding, matching
// the keys passed to TryUpdate.
// The value bytes must not be modified by the caller.
// If the trie is empty, both return values are nil.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryGetBestRightKeyAndValue() ([]byte, []byte, error) {
	key, value, newroot, didResolve, err := t.tryGetBestRightKeyAndValue(t.root, []byte{})
	if err == nil && didResolve {
		t.root = newroot
	}
	if err != nil {
		return nil, nil, err
	}
	if len(key) == 0 {
		return nil, nil, nil
	}
	return hexToKeybytes(key), value, nil
}

// tryGetBestRightKeyAndValue returns the rightmost leaf of the subtrie below
// origNode, carrying the hex-encoded key path accumulated in prefix. It
// resolves hash nodes on the way down and links the loaded children back
// into the trie on the way up.
func (t *Trie) tryGetBestRightKeyAndValue(origNode node, prefix []byte) (key []byte, value []byte, newnode node, didResolve bool, err error) {
	switch n := (origNode).(type) {
	case nil:
		return nil, nil, nil, false, nil
	case *shortNode:
		switch v := n.Val.(type) {
		case valueNode:
			return append(prefix, n.Key...), v, n, false, nil
		default:
		}
		key, value, newnode, didResolve, err = t.tryGetBestRightKeyAndValue(n.Val, append(prefix, n.Key...))
		if err == nil && didResolve {
			n = n.copy()
			n.Val = newnode
		}
		return key, value, n, didResolve, err
	case *fullNode:
		for i := len(n.Children) - 1; i >= 0; i-- {
			if n.Children[i] == nil {
				continue
			}
			key, value, newnode, didResolve, err = t.tryGetBestRightKeyAndValue(n.Children[i], append(prefix, byte(i)))
			if err == nil && didResolve {
				n = n.copy()
				n.Children[i] = newnode
			}
			return key, value, n, didResolve, err
		}
		return nil, nil, n, false, nil
	case hashNode:
		child, err := t.resolveHash(n, nil)
		if err != nil {
			return nil, nil, n, true, err
		}
		key, value, newnode, _, err := t.tryGetBestRightKeyAndValue(child, prefix)
		return key, value, newnode, true, err
	default:
		return nil, nil, nil, false, fmt.Errorf("%T: invalid node: %v", origNode, origNode)
	}
}

// TryGetAllLeftKeyAndValue returns the keys and values of all leaves whose
// hex-encoded key is strictly less than the hex-encoded form of limit.
// The limit and the returned keys are in keybyte encoding, matching the
// keys passed to TryUpdate.
// The value bytes must not be modified by the caller.
// If no leaf matches, no keys or values are returned.
// If a node was not found in the database, a MissingNodeError is returned.
func (t *Trie) TryGetAllLeftKeyAndValue(limit []byte) ([][]byte, [][]byte, error) {
	hexLimit := keybytesToHex(limit)
	hexLimit = hexLimit[0 : len(hexLimit)-1] // strip trailing 0x10 terminator

	dataKeys, values, newroot, didResolve, err := t.tryGetAllLeftKeyAndValue(t.root, []byte{}, hexLimit)
	if err == nil && didResolve {
		t.root = newroot
	}
	if err != nil {
		return nil, nil, err
	}
	keys := [][]byte{}
	for _, data := range dataKeys {
		keys = append(keys, hexToKeybytes(data))
	}
	return keys, values, nil
}

// tryGetAllLeftKeyAndValue returns the keys and values of all leaves in the
// subtrie below origNode whose hex-encoded key is strictly less than limit,
// carrying the hex-encoded key path accumulated in prefix. It resolves hash
// nodes on the way down and links the loaded children back into the trie on
// the way up.
func (t *Trie) tryGetAllLeftKeyAndValue(origNode node, prefix []byte, limit []byte) (keys [][]byte, values [][]byte, newnode node, didResolve bool, err error) {
	switch n := (origNode).(type) {
	case nil:
		return nil, nil, nil, false, nil
	case valueNode:
		key := make([]byte, len(prefix))
		copy(key, prefix)
		if bytes.Compare(key, limit) < 0 {
			keys = append(keys, key)
			values = append(values, n)
		}
		return keys, values, n, false, nil
	case *shortNode:
		ks, vs, newnode, didResolve, err := t.tryGetAllLeftKeyAndValue(n.Val, append(prefix, n.Key...), limit)
		if err == nil && didResolve {
			n = n.copy()
			n.Val = newnode
		}
		return ks, vs, n, didResolve, err
	case *fullNode:
		for i := len(n.Children) - 1; i >= 0; i-- {
			if n.Children[i] == nil {
				continue
			}
			newPrefix := append(prefix, byte(i))
			if bytes.Compare(newPrefix, limit) > 0 {
				continue
			}
			allKeys, allValues, cn, didResolve, err := t.tryGetAllLeftKeyAndValue(n.Children[i], newPrefix, limit)
			if err != nil {
				return nil, nil, n, false, err
			}
			if didResolve {
				n = n.copy()
				n.Children[i] = cn
			}
			keys = append(keys, allKeys...)
			values = append(values, allValues...)
		}
		return keys, values, n, didResolve, err
	case hashNode:
		child, err := t.resolveHash(n, nil)
		if err != nil {
			return nil, nil, n, true, err
		}
		ks, vs, newnode, _, err := t.tryGetAllLeftKeyAndValue(child, prefix, limit)
		return ks, vs, newnode, true, err
	default:
		return nil, nil, nil, false, fmt.Errorf("%T: invalid node: %v", origNode, origNode)
	}
}
