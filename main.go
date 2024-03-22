package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var chat = NewChat()

func main() {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(w, r)
	})

	err := http.ListenAndServe("localhost:8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	for {
		messageType, raw, err := conn.ReadMessage()

		if err != nil {
			chat.CloseClient(conn)
			log.Println(err)
			return
		}

		var message Message

		if messageType != websocket.TextMessage {
			log.Println("invalid message type")
			continue
		}

		err = json.Unmarshal(raw, &message)
		if err != nil {
			log.Println(err)
			return
		}

		_ = chat.HandleMessage(conn, &message)
	}
}
