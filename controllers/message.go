package controllers

import (
	"github.com/astaxie/beego/logs"
	"github.com/gorilla/websocket"
)

type Message struct {
	receiver string
	msg      []byte
	conn     *websocket.Conn
}

var (
	K_Msgs = make(chan Message, 1024)
)

func init() {
	go _MsgHandler()
}

func _MsgHandler() {
	for {
		msg := <-K_Msgs
		_SendMessage(&msg)
	}
}

func _SendMessage(msg *Message) {
	if msg.conn != nil {
		logs.Debug("Send Message %s To: %s, conn: %p", string(msg.msg), msg.receiver, msg.conn)
		err := msg.conn.WriteMessage(websocket.TextMessage, msg.msg)
		if err != nil {
			logs.Error("Send Message Failed, Msg: %s, To: %s, conn: %p, Error: %s", string(msg.msg), msg.receiver, msg.conn, err.Error())
		}
	}
}
