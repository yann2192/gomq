/**
    Copyright (C) 2013 Yann GUIBET <yannguibet@gmail.com>
    See LICENSE for details.
**/

package gomq

import (
	"code.google.com/p/go.crypto/pbkdf2"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
)

// Contains the cryptographic context.
type _AES struct {
	ctx    cipher.Stream
	engine cipher.Block
}

// Encrypts or decrypts the input with AES.
func (aes *_AES) update(input []byte) []byte {
	buff := append([]byte(nil), input...)
	aes.ctx.XORKeyStream(buff, buff)
	return buff
}

// Returns the block size of the AES context.
func (aes *_AES) blockSize() int {
	return aes.engine.BlockSize()
}

// Creates and initializes a new AES context.
func newAES(key, iv []byte) (res *_AES, err error) {
	engine, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ctx := cipher.NewCTR(engine, iv)
	return &_AES{ctx: ctx, engine: engine}, nil
}

// Returns the block size of AES.
func blockSizeAES() int {
	return aes.BlockSize
}

// Computes the SHA256 digest of the input.
func _SHA256(input []byte) []byte {
	ctx := sha256.New()
	_, err := ctx.Write(input)
	if err != nil {
		return nil
	}
	return ctx.Sum([]byte(""))
}

// Generates cryptographically strong random bytes.
func _rand(size int) []byte {
	buff := make([]byte, size)
	_, err := rand.Read(buff)
	if err != nil {
		return nil
	}
	return buff
}

// Computes the HMAC-SHA256 digest of the input/key pair.
func _HMAC_SHA256(input, key []byte) []byte {
	ctx := hmac.New(sha256.New, key)
	_, err := ctx.Write(input)
	if err != nil {
		return nil
	}
	return ctx.Sum([]byte(""))
}

// Computes the PBKDF2-SHA256 digest of a password using 10000 iterations and
// the given salt, and return a 32-byte output. If no salt is given, a random
// 8-byte salt is generated. The salt is also returned.
func _PBKDF2_SHA256(password, salt []byte) ([]byte, []byte) {
	if salt == nil {
		salt = _rand(8)
	}
	return salt, pbkdf2.Key(password, salt, 10000, 32, sha256.New)
}
