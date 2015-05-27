package main

import (
	"CryNet"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	json "github.com/bitly/go-simplejson"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var users map[int]*websocket.Conn
var connections map[*websocket.Conn]int
var unknownConnections map[*websocket.Conn]time.Time

func SendWSMessage(conn *websocket.Conn, msg map[string]interface{}) {
	if err := conn.WriteJSON(&msg); err != nil {
		log.Println("Write : " + err.Error())
		return
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	if trace {
		log.Println("new connection from:", r.RemoteAddr)
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	unknownConnections[conn] = time.Now()
	if err != nil {
		log.Println(err)
		return
	}

	for {
		rec := map[string]interface{}{}
		if err = conn.ReadJSON(&rec); err != nil {
			if err.Error() == "EOF" {
				log.Println("EOF")
				return
			}

			uid, exists := connections[conn]
			if exists {
				unsubChan <- fmt.Sprintf("%06d", uid)
				delete(users, uid)
				delete(connections, conn)
			}
			if trace {
				log.Println("Read error: "+err.Error(), ", client disconnected")
			}

			return
		}

		if trace {
			log.Println("websocket got message:", rec)
		}
		act := rec["act"]
		switch act {
		case "login":
			uid := int(rec["uid"].(float64))
			users[uid] = conn
			connections[conn] = uid
			message := make(map[string]interface{})
			message["errcode"] = CryNet.SUCCESS
			SendWSMessage(conn, message)

			subChan <- fmt.Sprintf("%06d", uid)
			break
		case "chat":
			atomic.AddInt32(&websocketMsgGotCount, 1)
			recChan <- rec
			break
		default:
			log.Println("unknown message:", rec)
		}
	}
}

func websocket_sender() {
	for {
		select {
		case jsonMsg := <-sndChan:
			if snd, err := json.NewJson([]byte(jsonMsg)); err == nil {
				toId, _ := snd.Get("to_id").Int()
				conn := users[toId]
				if conn != nil {
					if err = conn.WriteJSON(snd); err != nil {
						log.Println("websocket send message error:", err)
						break
					} else {
						websocketMsgSentCount++
						if trace {
							log.Println("websocket sent message:", toId, snd)
						}
					}
				}
			} else {
				log.Println("error message:", jsonMsg)
			}
		}
	}
}

func StartWebsocketServer() {
	users = make(map[int]*websocket.Conn)
	connections = make(map[*websocket.Conn]int)
	unknownConnections = make(map[*websocket.Conn]time.Time)

	go websocket_sender()

	http.HandleFunc("/session", serveWs)
	var addr = flag.String("addr", fmt.Sprintf(":%d", websocketPort), "http service address")
	log.Println("websocket server started", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
