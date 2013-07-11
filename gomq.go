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
	"sync"
	"syscall"
	"time"
)

const (
	// Export ØMQ socket type.
	PULL = zmq.PULL
	PUSH = zmq.PUSH

	_SALT = "\x0c\x199\xe5yn\xe8\xa1" // SALT used to derive the master key
)

// Contains information about a GOMQ connection. A GOMQ connection can handle
// many hosts.
type _ConnectionInfo struct {
	Host *list.List
	Type zmq.SocketType
	Sock *zmq.Socket
}

// Creates and initializes a new _ConnectionInfo.
func newConnectionInfo(host string, _type zmq.SocketType) *_ConnectionInfo {
	res := &_ConnectionInfo{Type: _type, Sock: nil}
	res.Host = list.New()
	res.Host.PushBack(host)
	return res
}

// Type to describe the argument of a GOMQ task.
type Args interface{}

// Type to describe a GOMQ task.
type Pfunc func(Args)

// Contains all information for launching a task.
type _Message struct {
	Job      string `codec:"job"`
	UUID     string `codec:"uuid"` // Not used yet.
	Params   Args   `codec:"params"`
	Priority uint   `codec:"priority"` // Not used yet.
}

// Creates and initializes a new _Message.
func newMessage(job, uuid string, params Args, priority uint) *_Message {
	return &_Message{job, uuid, params, priority}
}

// GOMQ structure.
type GOMQ struct {
	uuid        string                      // To identify a GOMQ instance. Not used yet.
	context     *zmq.Context                // A ØMQ context.
	jobs        map[string]Pfunc            // Contains the defined task.
	connections map[string]*_ConnectionInfo // Contains the created connection.
	key         []byte                      // Used for encryption.
	lock        sync.WaitGroup              // Used for waiting for running goroutine.
	run         bool                        // Used for the principal loop.
}

// Creates and initializes a new GOMQ instance identified by the given uuid.
func NewGOMQ(uuid string) *GOMQ {
	runtime.GOMAXPROCS(runtime.NumCPU())
	res := &GOMQ{uuid: uuid}
	res.jobs = map[string]Pfunc{}
	res.connections = map[string]*_ConnectionInfo{}
	res.context, _ = zmq.NewContext()
	res.key = nil
	return res
}

// Creates a new connection with the host and identified by the given name.
func (self *GOMQ) CreateConnection(name, host string, sock_type zmq.SocketType) {
	if self.connections[name] == nil {
		self.connections[name] = newConnectionInfo(host, sock_type)
	} else {
		self.connections[name].Host.PushBack(host)
	}
}

// Defines a new job identified by the string job.
func (self *GOMQ) AddJob(job string, action Pfunc) {
	self.jobs[job] = action
}

// Defines the master key using for encryption.
func (self *GOMQ) SetMasterKey(key []byte) {
	_, self.key = _PBKDF2_SHA256(key, []byte(_SALT))
}

// Sends a job on the connection identified by connection_name to execute the
// task identified by the string job which will take params as argument.
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
	msg := newMessage(job, uuid, params, 3)
	buff, err := encodeMessage(msg)
	if err != nil {
		return err
	}
	if self.key != nil {
		buff, err = self.encrypt(buff)
		if err != nil {
			return err
		}
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

// This loop will listen to all incoming connections on the given host.
// It will receive incoming jobs, and launch them.
func (self *GOMQ) Loop(host string, sock_type zmq.SocketType) error {
	self.AddTask()
	defer self.FreeTask()

	s, err := self.context.NewSocket(sock_type)
	if err != nil {
		return err
	}
	err = s.Bind(host)
	if err != nil {
		return err
	}

	// Receive kill signals.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, os.Signal(syscall.SIGTERM))
	go func() {
		<-c
		self.run = false
		signal.Stop(c)
	}()

	self.run = true
	for self.run {
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

// Closes all opens connections.
func (self *GOMQ) Close() {
	for _, sock_infos := range self.connections {
		sock_infos.Sock.Close()
		sock_infos.Sock = nil
	}
	self.context.Close()
}

// Registers a task. Wait() wait all registered tasks.
func (self *GOMQ) AddTask() {
	self.lock.Add(1)
}

// Unregisters a task.
func (self *GOMQ) FreeTask() {
	self.lock.Done()
}

// Waits for all registered tasks.
func (self *GOMQ) Wait() {
	self.lock.Wait()
}

// Handles received messages and executes the associated task.
func (self *GOMQ) handle(buff []byte) {
	var err error
	if self.key != nil {
		buff, err = self.decrypt(buff)
		if err != nil {
			log.Println("GOMQ:handle:decrypt", err)
			return
		}
	}
	msg, err := decodeMessage(buff)
	if err != nil {
		log.Println("GOMQ:handle:decodeMessage", err)
	} else {
		job := self.getJob(msg.Job)
		job(msg.Params)
	}
}

// Retrieves the job associated with the given string.
func (self *GOMQ) getJob(job string) Pfunc {
	return self.jobs[job]
}

// Creates a ØMQ socket (if it doesn't exist) with the given socket info.
func (self *GOMQ) createSock(sock_infos *_ConnectionInfo) (*zmq.Socket, error) {
	if sock_infos.Sock == nil {
		sock, err := self.context.NewSocket(sock_infos.Type)
		if err != nil {
			return nil, err
		}
		// Connect to each host.
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

// Encrypts the given buffer and generates an HMAC fingerprint.
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

// Decrypts the given buffer and verifies the HMAC fingerprint.
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
