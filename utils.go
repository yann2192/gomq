/**
    Copyright (C) 2013 Yann GUIBET <yannguibet@gmail.com>
    See LICENSE for details.
**/

package gomq

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"io"
)

func EncodeMessage(data *Message) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(*data)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func DecodeMessage(data []byte) (*Message, error) {
	var buffer bytes.Buffer
	res := new(Message)
	buffer.Write(data)
	decoder := gob.NewDecoder(&buffer)
	err := decoder.Decode(res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Take here : http://play.golang.org/p/4FkNSiUDMg
// newUUID generates a random UUID according to RFC 4122
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
