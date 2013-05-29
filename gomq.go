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
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

const (
	PULL  = zmq.PULL
	PUSH  = zmq.PUSH
	_SALT = "\x0c\x199\xe5yn\xe8\xa1"
)

type _ConnectionInfo struct {
	Host *list.List
	Type zmq.SocketType
	Sock *zmq.Socket
}

func newConnectionInfo(host string, _type zmq.SocketType) *_ConnectionInfo {
	res := &_ConnectionInfo{Type: _type, Sock: nil}
	res.Host = list.New()
	res.Host.PushBack(host)
	return res
}

type Args interface{}
type Pfunc func(Args)

type _Message struct {
	Job,
	UUID string
	Params   Args
	Priority uint
}

func newMessage(job, uuid string, params Args, priority uint) *_Message {
	return &_Message{job, uuid, params, priority}
}

type GOMQ struct {
	uuid        string
	context     *zmq.Context
	jobs        map[string]Pfunc
	connections map[string]*_ConnectionInfo
	key         []byte
	Run         bool
}

func NewGOMQ(uuid string) *GOMQ {
	runtime.GOMAXPROCS(runtime.NumCPU())
	res := &GOMQ{uuid: uuid}
	res.jobs = map[string]Pfunc{}
	res.connections = map[string]*_ConnectionInfo{}
	res.context, _ = zmq.NewContext()
	res.key = nil
	return res
}

func (self *GOMQ) createSock(sock_infos *_ConnectionInfo) (*zmq.Socket, error) {
	if sock_infos.Sock == nil {
		sock, err := self.context.NewSocket(sock_infos.Type)
		if err != nil {
			return nil, err
		}
		for e := sock_infos.Host.Front(); e != nil; e = e.Next() {
			err := sock.Connect(e.Value.(string))
			if err != nil {
				return nil, err
			}
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
	sock, err := self.createSock(sock_infos)
	if err != nil {
		return err
	}
	msg := newMessage(job, uuid, params, 0)
	buff, err := encodeMessage(msg)
	if err != nil {
		return err
	}
	buff, err = self.encrypt(buff)
	if err != nil {
		return err
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
	buff, err := self.decrypt(buff)
	if err != nil {
		log.Println("GOMQ:handle:decrypt", err)
		return
	}
	msg, err := decodeMessage(buff)
	if err != nil {
		log.Println("GOMQ:handle:decodeMessage", err)
	} else {
		job := self.getJob(msg.Job)
		job(msg.Params)
	}
}

func (self *GOMQ) Loop(host string, sock_type zmq.SocketType) error {
	s, err := self.context.NewSocket(sock_type)
	if err != nil {
		return err
	}
	err = s.Bind(host)
	if err != nil {
		return err
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, os.Signal(syscall.SIGTERM))
	go func() {
		<-c
		self.Run = false
		signal.Stop(c)
	}()

	self.Run = true
	for self.Run {
		buff, err := s.RecvMultipart(zmq.SNDMORE)
		if err != nil {
			log.Println("GOMQ:Loop:RecvMultipart", err)
			time.Sleep(time.Millisecond)
			continue
		}
		go self.handle(bytes.Join(buff, []byte(nil)))
	}

	s.Close()
	log.Println("GOMQ:Loop:Exiting")
	return nil
}

func (self *GOMQ) AddJob(job string, action Pfunc) {
	self.jobs[job] = action
}

func (self *GOMQ) getJob(job string) Pfunc {
	return self.jobs[job]
}

func (self *GOMQ) SetMasterKey(key []byte) {
	_, self.key = _PBKDF2_SHA256(key, []byte(_SALT))
}

func (self *GOMQ) encrypt(data []byte) ([]byte, error) {
	var buffer bytes.Buffer
	if self.key == nil {
		return nil, errors.New("GOMQ:encrypt:Master Key is not define")
	}
	iv := _rand(blockSizeAES())
	_, err := buffer.Write(iv)
	if err != nil {
		return nil, err
	}
	ctx, err := newAES(self.key, iv)
	if err != nil {
		return nil, err
	}
	ciphertext := ctx.update(data)
	_, err = buffer.Write(ciphertext)
	if err != nil {
		return nil, err
	}
	hmac := _HMAC_SHA256(data, self.key)
	_, err = buffer.Write(hmac)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (self *GOMQ) decrypt(buff []byte) ([]byte, error) {
	if self.key == nil {
		return nil, errors.New("GOMQ:encrypt:Master Key is not define")
	}
	length := len(buff)
	buffer := bytes.NewBuffer(buff)
	iv := make([]byte, blockSizeAES())
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
	ctx, err := newAES(self.key, iv)
	if err != nil {
		return nil, err
	}
	plaintext := ctx.update(data)
	hmac2 := _HMAC_SHA256(plaintext, self.key)
	if !bytes.Equal(hmac, hmac2) {
		return nil, errors.New("GOMQ:decrypt:HMAC check fail")
	}
	return plaintext, nil
}

func (self *GOMQ) Close() {
	for _, sock_infos := range self.connections {
		sock_infos.Sock.Close()
		sock_infos.Sock = nil
	}
	self.context.Close()
}
