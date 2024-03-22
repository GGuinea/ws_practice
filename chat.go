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

type Chat struct {
	mutex   sync.RWMutex // consider rwmutex
	rooms   map[string]*chatRoom
	clients map[*websocket.Conn]*client
}

func NewChat() *Chat {
	return &Chat{
		rooms:   make(map[string]*chatRoom),
		clients: make(map[*websocket.Conn]*client),
	}
}

func (c *Chat) CloseClient(conn *websocket.Conn) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	client, ok := c.clients[conn]
	if !ok {
		return
	}
	client.close <- struct{}{}
	delete(c.clients, conn)
	for _, clientRoom := range client.rooms {
		for _, listener := range c.rooms[clientRoom].listeners {
			if listener.conn == conn {
				c.LeaveRoom(clientRoom, conn)
			}
		}
	}
}

func (c *Chat) HandleMessage(conn *websocket.Conn, wsMessgae *Message) error {
	switch wsMessgae.MessageType {
	case MessageTypeRegister:
		err := c.RegisterNewWsClient(conn, wsMessgae.Data)
		if err != nil {
			resMessage := fmt.Sprintf("error registering user with name %s", wsMessgae.Data)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
				return err
			}
			return err
		}

		resMessage := fmt.Sprintf("registered user with name %s", wsMessgae.Data)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
			return err
		}
	case MessageTypeCreateRoom:
		err := c.CreateRoom(wsMessgae.RoomName)
		if err != nil {
			resMessage := fmt.Sprintf("error creating room with name %s", string(wsMessgae.RoomName))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
				return err
			}
			return err
		}

		resMessage := fmt.Sprintf("created room with name %s", string(wsMessgae.RoomName))
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
		err := c.JoinRoom(wsMessgae.RoomName, conn)
		if err != nil {
			log.Println(err)
			resMessage := fmt.Sprintf("error joining room with name %s", string(wsMessgae.RoomName))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
				return err
			}
			return err
		}
		resMessage := fmt.Sprintf("joined room with name %s", string(wsMessgae.RoomName))
		if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
			log.Println(err)
			return err
		}
	case MessageTypeLeaveRoom:
		err := c.LeaveRoom(wsMessgae.RoomName, conn)
		if err != nil {
			resMessage := fmt.Sprintf("error leaving room with name %s", string(wsMessgae.RoomName))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
				return err
			}
			return err
		}
		resMessage := fmt.Sprintf("left room with name %s", string(wsMessgae.RoomName))
		if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
			return err
		}
	case MessageTypeMessage:
		err := c.PublishMessage(wsMessgae.RoomName, wsMessgae.Data)
		if err != nil {
			resMessage := fmt.Sprintf("error publishing message to room with name %s", string(wsMessgae.RoomName))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
				return err
			}
			return err
		}
		resMessage := fmt.Sprintf("published message to room with name %s", string(wsMessgae.RoomName))
		if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
			return err
		}
	default:
		resMessage := fmt.Sprintf("invalid message type %s", wsMessgae.MessageType)
		if err := conn.WriteMessage(websocket.TextMessage, []byte(resMessage)); err != nil {
			return err
		}
		return fmt.Errorf("Invalid message type")
	}

	return nil
}

func (c *Chat) RegisterNewWsClient(conn *websocket.Conn, userName string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	_, ok := c.clients[conn]

	if ok {
		return fmt.Errorf("Already exists")
	}

	client := client{conn: conn, outMsgChan: make(chan []byte, 20), name: userName, close: make(chan struct{})}
	go client.writePump()

	c.clients[conn] = &client

	return nil
}

func (c *Chat) CreateRoom(roomName string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	_, ok := c.rooms[roomName]

	if ok {
		return fmt.Errorf("Already exists")
	}

	room := chatRoom{}

	c.rooms[roomName] = &room

	return nil
}

func (c *Chat) JoinRoom(roomName string, conn *websocket.Conn) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	room, ok := c.rooms[roomName]

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
	client.rooms = append(client.rooms, roomName)

	return nil
}

func (c *Chat) LeaveRoom(roomName string, conn *websocket.Conn) error {
	c.mutex.Lock()
	room, ok := c.rooms[roomName]
	c.mutex.Unlock()

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

func (c *Chat) PublishMessage(roomName string, message string) error {
	c.mutex.RLock()
	room, ok := c.rooms[roomName]
	c.mutex.RUnlock()

	if !ok {
		return fmt.Errorf("Room not found")
	}

	for _, listener := range room.listeners {
		select {
		case listener.outMsgChan <- []byte(message):
		default:
			log.Println("Cannot publish for: ", listener.name)
		}
	}

	return nil
}

func (c *Chat) ListRooms() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	rooms := make([]string, 0, len(c.rooms))

	for name := range c.rooms {
		rooms = append(rooms, name)
	}

	return rooms
}
