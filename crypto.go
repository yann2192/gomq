/**
    Copyright (C) 2013 Yann GUIBET <yannguibet@gmail.com>
    See LICENSE for details.
**/

package gomq

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"gomq/crypto/pbkdf2"
)

type AES struct {
	ctx    cipher.Stream
	engine cipher.Block
}

func (aes *AES) Update(input []byte) []byte {
	buff := append([]byte(nil), input...)
	aes.ctx.XORKeyStream(buff, buff)
	return buff
}

func (aes *AES) BlockSize() int {
	return aes.engine.BlockSize()
}

func NewAES(key, iv []byte) (res *AES, err error) {
	engine, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	ctx := cipher.NewCTR(engine, iv)
	return &AES{ctx: ctx, engine: engine}, nil
}

func BlockSizeAES() int {
	return aes.BlockSize
}

func SHA256(input []byte) []byte {
	ctx := sha256.New()
	_, err := ctx.Write(input)
	if err != nil {
		return nil
	}
	return ctx.Sum([]byte(""))
}

func Rand(size int) []byte {
	buff := make([]byte, size)
	_, err := rand.Read(buff)
	if err != nil {
		return nil
	}
	return buff
}

func HMAC_SHA256(input, key []byte) []byte {
	ctx := hmac.New(sha256.New, key)
	_, err := ctx.Write(input)
	if err != nil {
		return nil
	}
	return ctx.Sum([]byte(""))
}

func PBKDF2_SHA256(password, salt []byte) ([]byte, []byte) {
	if salt == nil {
		salt = Rand(8)
	}
	return salt, pbkdf2.Key(password, salt, 10000, 32, sha256.New)
}
