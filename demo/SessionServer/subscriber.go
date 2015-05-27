package main

import (
	"fmt"
	"log"

	zmq "github.com/pebbe/zmq4"
)

func StartSubscriber() {
	worker, _ := zmq.NewSocket(zmq.SUB)
	defer worker.Close()
	worker.Bind(fmt.Sprintf("tcp://*:%d", subscriberPort))
	log.Println("subscriber prepared")

	go func() {
		var uid string
		for {
			select {
			case uid = <-subChan:
				worker.SetSubscribe(uid)
				clientLoginedCount++
				if trace {
					log.Println("subscribed", uid)
				}
			case uid = <-unsubChan:
				worker.SetUnsubscribe(uid)
				if trace {
					log.Println("unsubscribed", uid)
				}
				clientLoginedCount--
			}
		}
	}()
	var msg []string
	var err error
	for {
		msg, err = worker.RecvMessage(0)
		if err != nil {
			log.Println("RecvMessage error:", err)
			break
		} else {
			if trace {
				log.Println("subscriber got message:", msg)
			}
			subscribedMsgCount++
			jsonMsg := msg[1]
			sndChan <- jsonMsg
		}
	}
}
