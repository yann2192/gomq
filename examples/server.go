package main

import (
	"gomq"
	"log"
	"time"
)

var h *gomq.GOMQ

func a(args gomq.Args) {
	log.Println(">", args.(string), "<")
	h.SendLocalJob("test2", "Bye !")
	time.Sleep(time.Second)
}

func b(args gomq.Args) {
	log.Println(">>", args.(string), "<<")
	time.Sleep(time.Second)
}

func Server() {
	h = gomq.NewGOMQ("test")
	h.SetMasterKey([]byte("test"))
	h.AddJob("test", a)
	h.AddJob("test2", b)
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
