package main

import (
	"flag"
	"log"
	"time"
)

var trace bool = false
var dealtCount int = 0
var publishedCount int = 0

const (
	DEALER_PORT    = 5670
	PUBLISHER_PORT = 5671
)

func init() {
	flag.BoolVar(&trace, "t", false, "Enable verbose output")
}

func main() {
	flag.Parse()

	StartWorkers()
	StartDiscovery()
	log.Println("Chat server started.")

	for {
		if !trace {
			log.Printf("Dealt %d, Published %d\n", dealtCount, publishedCount)
		}
		time.Sleep(time.Second)
	}
}
