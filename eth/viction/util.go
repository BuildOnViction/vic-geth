package viction

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
)

// Decrypt encrypted data using AES CFB mode,
func DecryptAesCfb(key []byte, cryptoText string) (string, error) {
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext), nil
}

// Decrypt randomize using secret and opening pair.
func DecryptRandomize(secrets [][32]byte, opening [32]byte) (int64, error) {
	if len(secrets) > 0 {
		result := int64(0)
		for _, secret := range secrets {
			ciphr := bytes.TrimLeft(secret[:], "\x00")
			text, err := DecryptAesCfb(opening[:], string(ciphr))
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
