/**
    Copyright (C) 2013 Yann GUIBET <yannguibet@gmail.com>
    See LICENSE for details.
**/

/*
A fast and lightweight distributed Task Queue using ØMQ.

GOMQ provide an easy, fast and secure way to send order on various network
schemas to execute task efficiently on remote hosts.

GOMQ contains defined task identified by a string. It can bind like
a daemon or connect to others daemons (or both). GOMQ use only encrypted
connections using AES. All incoming or outcoming connections are ØMQ connections
so a connection must only send (using gomq.PUSH) or receive (using gomq.PULL).
A connection can be connected to multiple hosts.

Daemon example:

    package main

    import (
        "gomq"
        "log"
        "time"
    )

    var _GOMQ *gomq.GOMQ

    func a(b gomq.Args) {
        _GOMQ.AddTask()
        defer _GOMQ.FreeTask()
        log.Println(">", b.(string), "<")
        time.Sleep(time.Second)
    }

    func Server() {
        _GOMQ = gomq.NewGOMQ("daemon")
        _GOMQ.SetMasterKey([]byte("test"))
        _GOMQ.AddJob("test", a)
        err := _GOMQ.Loop("tcp://127.0.0.1:6666", gomq.PULL)
        if err != nil {
            log.Println(err)
        }
        _GOMQ.Close()
        _GOMQ.Wait()
    }

    func main() {
        Server()
    }

Client example:

    package main

    import (
        "gomq"
        "log"
    )

    func Client() {
        h := gomq.NewGOMQ("client")
        h.SetMasterKey([]byte("test"))
        h.CreateConnection("todaemon", "tcp://127.0.0.1:6666", gomq.PUSH)
        err := h.SendJob("todaemon", "test", "HelloWorld!")
        if err != nil {
            log.Println("Client:SendJob", err)
        }
        h.Close()
    }

    func main() {
        Client()
    }



Requirements :

    * gozmq (https://github.com/alecthomas/gozmq)
    * pbkdf2 (https://code.google.com/p/go.crypto/pbkdf2)
*/
package gomq
