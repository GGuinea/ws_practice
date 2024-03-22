package main


const (
	MessageTypeCreateRoom = "createRoom"
	MessageTypeJoinRoom   = "joinRoom"
	MessageTypeLeaveRoom  = "leaveRoom"
	MessageTypeMessage    = "message"
	MessageTypeListRooms  = "listRooms"
	MessageTypeRegister   = "register"
)

type Message struct {
	MessageType string `json:"messageType"`
	UserName    string `json:"userName"`
	Data        string `json:"data"`
	RoomName    string `json:"roomName"`
}
