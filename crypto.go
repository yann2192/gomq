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

// Contain cryptographic context.
type _AES struct {
	ctx    cipher.Stream
	engine cipher.Block
}

// Encrypt for decrypt input.
func (aes *_AES) update(input []byte) []byte {
	buff := append([]byte(nil), input...)
	aes.ctx.XORKeyStream(buff, buff)
	return buff
}

// Return block size of the AES context.
func (aes *_AES) blockSize() int {
	return aes.engine.BlockSize()
}

// Create and initialize a new AES context.
func newAES(key, iv []byte) (res *_AES, err error) {
	engine, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ctx := cipher.NewCTR(engine, iv)
	return &_AES{ctx: ctx, engine: engine}, nil
}

// Return aes block size.
func blockSizeAES() int {
	return aes.BlockSize
}

// Compute the input with SHA256.
func _SHA256(input []byte) []byte {
	ctx := sha256.New()
	_, err := ctx.Write(input)
	if err != nil {
		return nil
	}
	return ctx.Sum([]byte(""))
}

// Generate cryptographic random data.
func _rand(size int) []byte {
	buff := make([]byte, size)
	_, err := rand.Read(buff)
	if err != nil {
		return nil
	}
	return buff
}

// Compute input and key using HMAC SHA256.
func _HMAC_SHA256(input, key []byte) []byte {
	ctx := hmac.New(sha256.New, key)
	_, err := ctx.Write(input)
	if err != nil {
		return nil
	}
	return ctx.Sum([]byte(""))
}

// Compute password and salt using PBKDF2 SHA256 with 10000 iterations and
// return an 32 bytes length ouput.
func _PBKDF2_SHA256(password, salt []byte) ([]byte, []byte) {
	if salt == nil {
		salt = _rand(8)
	}
	return salt, pbkdf2.Key(password, salt, 10000, 32, sha256.New)
}
