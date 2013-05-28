package main

import (
	"gomq"
	"log"
)

func a(args gomq.Args) {
	log.Println(">", args.(string), "<")
}

func Server() {
	h := gomq.NewGOMQ("test")
	h.SetMasterKey([]byte("test"))
	h.AddJob("test", a)
	err := h.Loop("tcp://127.0.0.1:6666", gomq.PULL)
	if err != nil {
		log.Println(err)
	}
	h.Close()
}

func main() {
	Server()
}
