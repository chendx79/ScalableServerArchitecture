// ChatClient.go
package main

import (
	"CryNet"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

var trace bool = false

var (
	fromIdMin int = 0
	fromIdMax int = 999
	toIdMin   int = 0
	toIdMax   int = 999
)

var (
	users       map[int]*websocket.Conn
	connections map[*websocket.Conn]int
)

var (
	msgSentCount int = 0
	msgGotCount  int = 0
)

var wsURL string = "ws://127.0.0.1:15001/session"

func createWSClient(urlString string) (*websocket.Conn, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}

	rawConn, err := net.Dial("tcp", u.Host)
	if err != nil {
		return nil, err
	}

	wsHeaders := http.Header{
		//"Origin": {urlString},
		"Sec-WebSocket-Extensions": {"permessage-deflate; client_max_window_bits, x-webkit-deflate-frame"},
	}

	wsConn, resp, err := websocket.NewClient(rawConn, u, wsHeaders, 1024, 1024)
	if err != nil {
		return nil, fmt.Errorf("websocket.NewClient Error: %s\nResp:%+v", err, resp)
	}
	return wsConn, nil
}

func init() {
	flag.BoolVar(&trace, "t", false, "Enable verbose output")
}

func main() {
	log.Println("Stress test starts")

	flag.Parse()
	if flag.NArg() == 5 {
		fromIdMin = CryNet.StrToInt(flag.Arg(0))
		fromIdMax = CryNet.StrToInt(flag.Arg(1))
		toIdMin = CryNet.StrToInt(flag.Arg(2))
		toIdMax = CryNet.StrToInt(flag.Arg(3))
		wsURL = flag.Arg(4)
	} else {
		log.Printf("Using default arguments")
	}

	rand.Seed(time.Now().UnixNano())

	users = make(map[int]*websocket.Conn)
	connections = make(map[*websocket.Conn]int)

	go func() {
		for {
			if !trace {
				log.Printf("Got %d messages, Sent %d messages\n", msgGotCount, msgSentCount)
			}
			time.Sleep(time.Second)
		}
	}()

	var snd map[string]interface{}
	for i := fromIdMin; i <= fromIdMax; i++ {
		conn, err := createWSClient(wsURL)
		if err != nil {
			log.Println(err)
			return
		}

		snd = map[string]interface{}{}
		snd["act"] = "login"
		snd["uid"] = i
		conn.WriteJSON(snd)
		msgSentCount++
		if trace {
			log.Println(i, "sent", snd)
		}

		users[i] = conn
		connections[conn] = i

		go func(conn *websocket.Conn, uid int) {
			var rec interface{}
			for {
				if err := conn.ReadJSON(&rec); err == nil {
					if trace {
						log.Println(uid, "got", rec)
					}
					msgGotCount++
				} else {
					time.Sleep(time.Millisecond)
				}
			}
		}(conn, i)
	}

	log.Println("clients created")
	time.Sleep(time.Second * 2)
	msgGotCount = 0
	msgSentCount = 0
	log.Println("start to send message")

	var conn *websocket.Conn
	for {
		for i := fromIdMin; i <= fromIdMax; i++ {
			conn = users[i]
			snd = map[string]interface{}{}
			snd["act"] = "chat"
			snd["from_id"] = i
			snd["to_id"] = rand.Intn(toIdMax-toIdMin+1) + toIdMin
			conn.WriteJSON(snd)
			msgSentCount++
			if trace {
				log.Println(i, "sent", snd)
				time.Sleep(time.Second * 5)
			} else {
				time.Sleep(time.Millisecond * 5)
			}
		}
		time.Sleep(time.Millisecond * 100)
	}
}
