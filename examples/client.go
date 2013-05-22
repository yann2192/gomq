package main

import (
	"gomq"
	"log"
)

func Client() {
	h := gomq.NewGOMQ("test2")
	h.SetMasterKey([]byte("test"))
	h.CreateConnection("test", "tcp://127.0.0.1:6666", gomq.PUSH)
	err := h.SendJob("test", "test", "you")
	if err != nil {
		log.Println("Client:SendJob", err)
	}
	err = h.SendJob("test", "test", "mother")
	if err != nil {
		log.Println("Client:SendJob", err)
	}
	err = h.SendJob("test", "test", "fucker")
	if err != nil {
		log.Println("Client:SendJob", err)
	}
	h.Close()
}

func main() {
	Client()
}
