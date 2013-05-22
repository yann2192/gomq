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
	h := gomq.NewGOMQ("test")
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
