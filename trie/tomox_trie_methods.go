package trie

import (
	"errors"
)

// TryGetBestLeftKeyAndValue returns the left most node under the root hash
func (t *Trie) TryGetBestLeftKeyAndValue() ([]byte, []byte, error) {
	it := t.NodeIterator(nil)
	for it.Next(true) {
		if it.Leaf() {
			key := it.LeafKey()
			return key, it.LeafBlob(), nil
		}
	}
	return nil, nil, errors.New("not found")
}

// TryGetBestRightKeyAndValue returns the right most node under the root hash (largest key)
// It is slower than left key search because it requires full traversal.
func (t *Trie) TryGetBestRightKeyAndValue() ([]byte, []byte, error) {
	it := t.NodeIterator(nil)
	var lastKey, lastVal []byte
	for it.Next(true) {
		if it.Leaf() {
			lastKey = it.LeafKey()
			lastVal = it.LeafBlob()
		}
	}
	if lastKey == nil {
		return nil, nil, errors.New("not found")
	}
	return lastKey, lastVal, nil
}

// TryGetAllLeftKeyAndValue gets all entries bounded by maxCount.
func (t *Trie) TryGetAllLeftKeyAndValue(maxCount int) ([][]byte, [][]byte, error) {
	it := t.NodeIterator(nil)
	var keys [][]byte
	var vals [][]byte
	count := 0
	for it.Next(true) {
		if it.Leaf() {
			keys = append(keys, it.LeafKey())
			vals = append(vals, it.LeafBlob())
			count++
			if count >= maxCount {
				break
			}
		}
	}
	if len(keys) == 0 {
		return nil, nil, errors.New("not found")
	}
	return keys, vals, nil
}
