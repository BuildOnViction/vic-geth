// Copyright 2025 The Viction Authors
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

package viction

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

// Decrypt encrypted data using AES CFB mode,
func DecryptAesCfb(ecrypted, key []byte) (string, error) {
	ciphr := make([]byte, len(ecrypted))
	copy(ciphr, ecrypted)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphr) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}
	iv := ciphr[:aes.BlockSize]
	ciphr = ciphr[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphr, ciphr)

	return fmt.Sprintf("%s", ciphr), nil
}

// Decrypt randomize using secret and opening pair.
func DecryptRandomize(secrets [][32]byte, opening [32]byte) (int64, error) {
	if len(secrets) > 0 {
		result := int64(0)
		for _, secret := range secrets {
			ciphr := bytes.TrimLeft(secret[:], "\x00")
			text, err := DecryptAesCfb(opening[:], ciphr)
			if err != nil {
				return -1, err
			}
			number, err := strconv.ParseInt(text, 10, 64)
			if err != nil {
				return -1, err
			}
			result = number
		}
		return result, nil
	}
	return -1, nil
}

// Generate a dynamic array from *start*, increase by *step* unit by *repeat* times.
func GenerateSequence(start, step, repeat int64) []int64 {
	s := make([]int64, repeat)
	v := start
	for i := range s {
		s[i] = v
		v += step
	}

	return s
}

// Cast []common.Hash
func CommonHashesToBytesArray(hashes []common.Hash) [][32]byte {
	bytesArr := make([][32]byte, len(hashes))
	for i, hash := range hashes {
		bytesArr[i] = hash
	}
	return bytesArr
}
