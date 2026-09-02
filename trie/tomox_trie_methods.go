// Copyright 2026 The Vic-geth Authors
package trie

import (
	"bytes"
	"fmt"
)

// TryGetBestLeftKeyAndValue returns the leftmost (smallest key) leaf in the trie.
//
// Returns (nil, nil, nil) when the trie is empty.
// Propagates any underlying database error (e.g. MissingNodeError) encountered
// while resolving hash nodes from disk.
//
// The returned key is in keybyte (non-hex) encoding, matching the original
// key passed to TryUpdate.
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

// TryGetBestRightKeyAndValue returns the rightmost (largest key) leaf in the trie.
//
// Returns (nil, nil, nil) when the trie is empty.
// Propagates any underlying database error encountered while resolving hash nodes.
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

// TryGetAllLeftKeyAndValue returns all leaves whose hex-encoded key is strictly
// less than the hex-encoded form of limit (limit[0 : len-1] in hex).
//
// The limit byte slice is in keybyte encoding (same as TryUpdate keys).
// Returns (nil, nil, nil) when no matching leaves exist.
// Propagates any underlying database error encountered while resolving hash nodes.
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
