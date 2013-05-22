/**
    Copyright (C) 2013 Yann GUIBET <yannguibet@gmail.com>
    See LICENSE for details.
**/

package gomq

import (
	"bytes"
	"container/list"
	"errors"
	zmq "github.com/alecthomas/gozmq"
	"log"
	"runtime"
)

const (
	PULL = zmq.PULL
	PUSH = zmq.PUSH
	SALT = "\x0c\x199\xe5yn\xe8\xa1"
)

type connectionInfo struct {
	Host *list.List
	Type zmq.SocketType
	Sock *zmq.Socket
}

func newConnectionInfo(host string, _type zmq.SocketType) *connectionInfo {
	res := &connectionInfo{Type: _type, Sock: nil}
	res.Host = list.New()
	res.Host.PushBack(host)
	return res
}

type Args interface{}
type Pfunc func(Args)

type Message struct {
	Job,
	UUID string
	Params   Args
	Priority uint
}

func newMessage(job, uuid string, params Args, priority uint) *Message {
	return &Message{job, uuid, params, priority}
}

type GOMQ struct {
	uuid        string
	context     *zmq.Context
	jobs        map[string]Pfunc
	connections map[string]*connectionInfo
	pool        chan byte
	key         []byte
	Run         bool
}

func NewGOMQ(uuid string) *GOMQ {
	res := &GOMQ{uuid: uuid}
	res.jobs = map[string]Pfunc{}
	res.connections = map[string]*connectionInfo{}
	res.context, _ = zmq.NewContext()
	res.pool = nil
	res.key = nil
	return res
}

func (self *GOMQ) Jobs() map[string]Pfunc {
	return self.jobs
}

func (self *GOMQ) CreatePool(size int) {
	self.pool = make(chan byte, size)
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func (self *GOMQ) createSock(sock_infos *connectionInfo) (*zmq.Socket, error) {
	if sock_infos.Sock == nil {
		sock, err := self.context.NewSocket(sock_infos.Type)
		if err != nil {
			return nil, err
		}
		for e := sock_infos.Host.Front(); e != nil; e = e.Next() {
			sock.Connect(e.Value.(string))
		}
		sock_infos.Sock = sock
		return sock, nil
	} else {
		return sock_infos.Sock, nil
	}
}

func (self *GOMQ) CreateConnection(name, host string, sock_type zmq.SocketType) {
	if self.connections[name] == nil {
		self.connections[name] = newConnectionInfo(host, sock_type)
	} else {
		self.connections[name].Host.PushBack(host)
	}
}

func (self *GOMQ) SendJob(connection_name, job string, params Args) error {
	uuid, err := newUUID()
	if err != nil {
		return err
	}
	sock_infos := self.connections[connection_name]
	sock, err2 := self.createSock(sock_infos)
	if err2 != nil {
		return err2
	}
	msg := newMessage(job, uuid, params, 0)
	buff, err3 := EncodeMessage(msg)
	if err3 != nil {
		return err3
	}
	buff, err3 = self.encrypt(buff)
	if err3 != nil {
		return err3
	}
	for i := 0; i < len(buff); i += 4096 {
		limit := i + 4096
		if limit > len(buff) {
			limit = len(buff)
		}
		err = sock.Send(buff[i:limit], zmq.SNDMORE)
		if err != nil {
			return err
		}
	}
	err = sock.Send([]byte(nil), 0)
	return err
}

func (self *GOMQ) handle(buff []byte) {
	defer func() {
		<-self.pool
	}()
	buff, err := self.decrypt(buff)
	if err != nil {
		log.Println("GOMQ:handle:decrypt", err)
		return
	}
	msg, err2 := DecodeMessage(buff)
	if err2 != nil {
		log.Println("GOMQ:handle:DecodeMessage", err2)
	} else {
		job := self.getJob(msg.Job)
		job(msg.Params)
	}
}

func (self *GOMQ) Loop(host string, sock_type zmq.SocketType) error {
	if self.pool == nil {
		return errors.New("GOMQ:Loop:Pool not created")
	}
	s, err := self.context.NewSocket(sock_type)
	if err != nil {
		return err
	}
	s.Bind(host)
	self.Run = true
	for self.Run {
		buff, err := s.RecvMultipart(zmq.SNDMORE)
		if err != nil {
			log.Println("GOMQ:Loop:RecvMultipart", err)
			continue
		}
		self.pool <- '0'
		go self.handle(bytes.Join(buff, []byte(nil)))
	}
	return nil
}

func (self *GOMQ) AddJob(job string, action Pfunc) {
	self.jobs[job] = action
}

func (self *GOMQ) getJob(job string) Pfunc {
	return self.jobs[job]
}

func (self *GOMQ) SetMasterKey(key []byte) {
	_, self.key = PBKDF2_SHA256(key, []byte(SALT))
}

func (self *GOMQ) encrypt(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	if self.key == nil {
		return nil, errors.New("GOMQ:encrypt:Master Key is not define")
	}
	iv := Rand(BlockSizeAES())
	_, err := buffer.Write(iv)
	if err != nil {
		return nil, err
	}
	ctx, err2 := NewAES(self.key, iv)
	if err2 != nil {
		return nil, err2
	}
	ciphertext := ctx.Update(data)
	_, err3 := buffer.Write(ciphertext)
	if err3 != nil {
		return nil, err3
	}
	hmac := HMAC_SHA256(data, self.key)
	_, err4 := buffer.Write(hmac)
	if err4 != nil {
		return nil, err4
	}
	return buffer.Bytes(), nil
}

func (self *GOMQ) decrypt(buff []byte) ([]byte, error) {
	if self.key == nil {
		return nil, errors.New("GOMQ:encrypt:Master Key is not define")
	}
	length := len(buff)
	buffer := bytes.NewBuffer(buff)
	iv := make([]byte, BlockSizeAES())
	hmac := make([]byte, 32)
	data := make([]byte, length-(len(iv)+len(hmac)))
	i, err := buffer.Read(iv)
	if err != nil || i < len(iv) {
		return nil, err
	}
	i, err = buffer.Read(data)
	if err != nil || i < len(iv) {
		return nil, err
	}
	i, err = buffer.Read(hmac)
	if err != nil || i < len(iv) {
		return nil, err
	}
	ctx, err2 := NewAES(self.key, iv)
	if err2 != nil {
		return nil, err2
	}
	plaintext := ctx.Update(data)
	hmac2 := HMAC_SHA256(plaintext, self.key)
	if !bytes.Equal(hmac, hmac2) {
		return nil, errors.New("GOMQ:decrypt:HMAC check fail")
	}
	return plaintext, nil
}
