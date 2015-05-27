package main

import (
	"demo/service"
	"flag"
	"log"
	"time"
)

var trace bool = false
var (
	recChan   = make(chan map[string]interface{}, 100000)
	sndChan   = make(chan string, 100000)
	subChan   = make(chan string, 100000)
	unsubChan = make(chan string, 100000)
)

var (
	clientLoginedCount int = 0
	routerMsgGotCount  int = 0
	routerMsgSentCount int = 0
	subscribedMsgCount int = 0
)
var (
	websocketMsgGotCount  int32 = 0
	websocketMsgSentCount int32 = 0
)

var (
	websocketPort  int = 15001
	routerPort     int = 5670
	subscriberPort int = 5671
)

func init() {
	flag.BoolVar(&trace, "t", false, "Enable verbose output")
}

func main() {
	flag.Parse()

	go StartDealer()
	go StartWebsocketServer()
	go StartSubscriber()

	srv := service.New(service.SESSION)
	srv.Start()

	log.Println("Session server started.")
	for {
		if !trace {
			log.Printf("Logined %d, Websocket got %d, Sent %d, Router got %d, sent %d, Subscribed %d\n", clientLoginedCount, websocketMsgGotCount, websocketMsgSentCount, routerMsgGotCount, routerMsgSentCount, subscribedMsgCount)
		}
		time.Sleep(time.Second * 1)
	}
}
