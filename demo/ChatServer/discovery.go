package main

import (
	"demo/service"
	"log"
)

var serverReg = make(map[string]bool)
var serverChan = make(chan string, 32)

func OnSessionServerFound(ip string) {
	log.Println("Found session server", ip)
	serverChan <- ip
}

func OnSessionServerLost(ip string) {
	log.Println("Lost session server", ip)
}

func StartDiscovery() {
	srv := service.New(service.CHAT)
	srv.Discover(service.SESSION, OnSessionServerFound, OnSessionServerLost)
	srv.Start()
}
