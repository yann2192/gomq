package main

import (
	"gomq"
	"log"
)

var _GOMQ *gomq.GOMQ

func a(args gomq.Args) {
	_GOMQ.AddWorker()
	defer _GOMQ.FreeWorker()
	log.Println(">", args.(string), "<")
}

func Server() {
	_GOMQ = gomq.NewGOMQ("test")
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
