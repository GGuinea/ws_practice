package main

type Chat struct {
	rooms []string
}

func (c *Chat) CreateRoom(name string) {
	c.rooms = append(c.rooms, name)
}
