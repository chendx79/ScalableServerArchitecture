package main

import (
	"demo/service"
	"fmt"
	"log"

	json "github.com/bitly/go-simplejson"
	zmq "github.com/pebbe/zmq4"
)

func construct(ipaddr string, port uint16) (url string) {
	log.Printf("ipaddr: %s", ipaddr)
	url = fmt.Sprintf("tcp://%s:%d", ipaddr, port)
	return
}

func send_to_publisher(publisher *zmq.Socket, msg string) error {
	var rec *json.Json
	var err error
	var to_id int

	rec, err = json.NewJson([]byte(msg))
	if err != nil {
		return err
	}

	if to_id, err = rec.Get("to_id").Int(); err != nil {
		return err
	}

	if _, err = publisher.SendMessage(fmt.Sprintf("%06d", to_id), msg); err == nil {
		publishedCount++
		if trace {
			log.Println("publisher sent message:", to_id, msg)
		}
	}

	return err
}

func chat_worker() {
	dealer, _ := zmq.NewSocket(zmq.DEALER)
	defer dealer.Close()

	publisher, _ := zmq.NewSocket(zmq.PUB)
	defer publisher.Close()

	poller := zmq.NewPoller()
	poller.Add(dealer, zmq.POLLIN)

	var err error
	var msg string
	for {
		sockets, _ := poller.Poll(service.PING_INTERVAL)
		for _, socket := range sockets {
			switch s := socket.Socket; s {
			case dealer:
				if msg, err = dealer.Recv(0); err == nil {
					dealtCount++

					if trace {
						log.Println("dealer got message", msg)
					}
				}

				err = send_to_publisher(publisher, msg)
				if err != nil {
					log.Println(err)
				}
			}
		}
		select {
		case ipaddr := <-serverChan:
			dealer_url := construct(ipaddr, DEALER_PORT)
			dealer.Connect(dealer_url)
			log.Printf("Dealer: Connected to session server: %s", dealer_url)

			publisher_url := construct(ipaddr, PUBLISHER_PORT)
			publisher.Connect(publisher_url)
			log.Printf("Publisher: Connected to session server: %s", publisher_url)
		default:
		}
	}
}

func StartWorkers() {
	go chat_worker()
}
