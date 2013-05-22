====
GOMQ
====

GOMQ, a fast and lightweight distributed Messaging/Task Queue using ØMQ and some
cryptography.

TODO
====
    * Documentation
    * Tests
    * More examples

Example
=======

Daemon
------
::

    package main

    import (
        "gomq"
        "log"
        "time"
    )

    func a(b gomq.Args) {
        log.Println(">", b.(string), "<")
        time.Sleep(time.Second)
    }

    func Server() {
        h := gomq.NewGOMQ("daemon")
        h.SetMasterKey([]byte("test"))
        h.AddJob("test", a)
        h.CreatePool(2)
        err := h.Loop("tcp://127.0.0.1:6666", gomq.PULL)
        if err != nil {
            log.Println(err)
        }
        h.Close()
    }

    func main() {
        Server()
    }

Client
------
::

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



Requirements
============
    * gozmq (https://github.com/alecthomas/gozmq)
