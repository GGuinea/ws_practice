package main

import (
	"log"

	"github.com/gorilla/websocket"
)

type client struct {
	conn       *websocket.Conn
	outMsgChan chan []byte
	name       string
	rooms      []string
	close      chan struct{}
}

func (c *client) writePump() {
	for {
		select {
		case message, ok := <-c.outMsgChan:
			if !ok {
				log.Println("error reading from channel")
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println(err)
				return
			}
		case <-c.close:
			return
		}
	}
}
