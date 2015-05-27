package main

import (
	"encoding/json"
	"fmt"
	"log"

	zmq "github.com/pebbe/zmq4"
)

func StartDealer() {
	dealer, _ := zmq.NewSocket(zmq.DEALER)
	defer dealer.Close()
	dealer.Bind(fmt.Sprintf("tcp://*:%d", routerPort))

	log.Println("dealer started.")
	var jsonString []byte
	for {
		select {
		case msg := <-recChan:
			routerMsgGotCount++
			jsonString, _ = json.Marshal(msg)
			if _, err := dealer.Send(string(jsonString), 0); err == nil {
				routerMsgSentCount++
				if trace {
					log.Println("dealer sent message:", msg)
				}
			}
		}
	}
}
