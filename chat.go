package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type chatRoom struct {
	listeners []*client
}

type client struct {
	conn       *websocket.Conn
	outMsgChan chan []byte
	name       string
}

func (c *client) writePump() {
	for {
		select {
		case message, ok := <-c.outMsgChan:
			log.Println("writing message", message)
			if !ok {
				log.Println("error reading from channel")
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

type Chat struct {
	mutex   sync.Mutex // consider rwmutex
	rooms   map[string]*chatRoom
	clients map[*websocket.Conn]*client
}

func NewChat() *Chat {
	return &Chat{
		rooms:   make(map[string]*chatRoom),
		clients: make(map[*websocket.Conn]*client),
	}
}

func writeMessage(conn *websocket.Conn, message []byte) error {
	return conn.WriteMessage(websocket.TextMessage, message)
}

func (c *Chat) HandleMessage(conn *websocket.Conn, mess *Message) error {

	switch mess.MessageType {
	case MessageTypeRegister:
		err := c.Register(conn, mess.UserName)
		if err != nil {
			resMessage := fmt.Sprintf("error registering user with name %s", mess.UserName)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
				return err
			}
			return err
		}

		resMessage := fmt.Sprintf("registered user with name %s", mess.UserName)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
			return err
		}
	case MessageTypeCreateRoom:
		err := c.CreateRoom(mess.Data)
		if err != nil {
			resMessage := fmt.Sprintf("error creating room with name %s", string(mess.Data))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
				return err
			}
			return err
		}

		resMessage := fmt.Sprintf("created room with name %s", string(mess.RoomName))
		if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
			return err
		}
	case MessageTypeListRooms:
		rooms := c.ListRooms()
		resMessage := fmt.Sprintf("rooms: %v", rooms)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
			return err
		}
	case MessageTypeJoinRoom:
		err := c.JoinRoom(mess.RoomName, conn)
		if err != nil {
			log.Println(err)
			resMessage := fmt.Sprintf("error joining room with name %s", string(mess.Data))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
				return err
			}
			return err
		}
		resMessage := fmt.Sprintf("joined room with name %s", string(mess.Data))
		if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
			log.Println(err)
			return err
		}
	case MessageTypeLeaveRoom:
		// c.LeaveRoom(mess.Data)
	case MessageTypeMessage:
		c.Message(mess.RoomName, []byte(mess.Data))
	default:
		return fmt.Errorf("Invalid message type")
	}

	return nil
}

func (c *Chat) Register(conn *websocket.Conn, name string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	_, ok := c.clients[conn]

	if ok {
		return fmt.Errorf("Already exists")
	}

	client := client{conn: conn, outMsgChan: make(chan []byte), name: name}
	go client.writePump()

	c.clients[conn] = &client

	return nil
}

func (c *Chat) CreateRoom(name string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	_, ok := c.rooms[name]

	if ok {
		return fmt.Errorf("Already exists")
	}

	room := chatRoom{}

	c.rooms[name] = &room

	return nil
}

func (c *Chat) JoinRoom(name string, conn *websocket.Conn) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	room, ok := c.rooms[name]

	if !ok {
		return fmt.Errorf("Room not found")
	}

	client, ok := c.clients[conn]

	if !ok {
		return fmt.Errorf("Client not found")
	}

	for _, listener := range room.listeners {
		if listener.conn == conn {
			return fmt.Errorf("Already joined")
		}
	}

	room.listeners = append(room.listeners, client)

	return nil
}

func (c *Chat) LeaveRoom(name string, conn *websocket.Conn) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	room, ok := c.rooms[name]

	if !ok {
		return fmt.Errorf("Room not found")
	}

	for i, listener := range room.listeners {
		if listener.conn == conn {
			room.listeners = append(room.listeners[:i], room.listeners[i+1:]...)
			break
		}
	}

	return nil
}

func (c *Chat) Message(name string, message []byte) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	room, ok := c.rooms[name]

	if !ok {
		return fmt.Errorf("Room not found")
	}

	for _, listener := range room.listeners {
		listener.outMsgChan <- message
	}

	return nil
}

func (c *Chat) ListRooms() []string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	rooms := make([]string, 0, len(c.rooms))

	for name := range c.rooms {
		rooms = append(rooms, name)
	}

	return rooms
}

func (c *Chat) Run() {
	for {

	}
}
