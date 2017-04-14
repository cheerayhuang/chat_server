package controllers

import (
	"chat_server/models"

	"fmt"
	"net/http"
	"time"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"

	"github.com/bitly/go-simplejson"
	"github.com/gorilla/websocket"
)

const (
	PARSE_JSON_ERR  = 1000
	MISS_PARAM_ERR  = 1100
	CMD_TYPE_ERR    = 1200
	PERMISSION_ERR  = 1300
	LOGIN_ERR       = 2000
	ADD_USER_ERR    = 3000
	DELETE_USER_ERR = 4000
)

const (
	WS_READ_BUFFER_SIZE  = 1024
	WS_WRITE_BUFFER_SIZE = 1024
)

const (
	WELCOMD_MSG = "Welcom new user \"%s\" to join chatting."
	BYEBYE_MSG  = "User \"%s\" left chatting."
)

var (
	ERR_REPLYS = map[int]string{
		PARSE_JSON_ERR:  "Message is NOT in JSON format.",
		MISS_PARAM_ERR:  "Miss parameters in JSON.",
		CMD_TYPE_ERR:    "Unknown command type.",
		PERMISSION_ERR:  "No user login or the user doesn't have permisson to exec this command.",
		LOGIN_ERR:       "Login failed. User does NOT exist or password is Wrong.",
		ADD_USER_ERR:    "Add user failed. Maybe user name is duplicated.",
		DELETE_USER_ERR: "Delete user failed.",
	}

	WS_CLOSE_ERROR = []int{
		websocket.CloseNormalClosure,
		websocket.CloseGoingAway,
		websocket.CloseProtocolError,
		websocket.CloseUnsupportedData,
		websocket.CloseNoStatusReceived,
		websocket.CloseAbnormalClosure,
		websocket.CloseInvalidFramePayloadData,
		websocket.ClosePolicyViolation,
		websocket.CloseMessageTooBig,
		websocket.CloseMandatoryExtension,
		websocket.CloseInternalServerErr,
		websocket.CloseServiceRestart,
		websocket.CloseTryAgainLater,
		websocket.CloseTLSHandshake,
	}
)

var (
	k_online_users = make(map[string]*websocket.Conn)
)

type ChatController struct {
	beego.Controller

	cur_cmd       string
	cur_user      string
	cur_user_type int
	ws            *websocket.Conn
	reply_json    *simplejson.Json
	body_json     *simplejson.Json
}

func (this *ChatController) ErrReply(err_code int) {
	j := simplejson.New()
	j.Set("type", this.cur_cmd)
	j.Set("code", err_code)
	j.Set("reason", ERR_REPLYS[err_code])

	this.Reply(j)
}

func (this *ChatController) Reply(j *simplejson.Json) {
	if this.ws != nil {
		data, err := j.MarshalJSON()
		if err != nil {
			logs.Error("MarshalJSON failed.")
		} else {
			err := this.ws.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				logs.Error("Command \"%s\" Response Faild. Error: ", err.Error())
			}
		}
	} else {
		logs.Error("Current connetion is lost.")
	}
}

func (this *ChatController) SendMsg(j *simplejson.Json, receivers []string) {
	data, err := j.MarshalJSON()
	if err != nil {
		logs.Error("SendMsg MarshalJSON failed. Error: ", err.Error())
		panic(err)
	}
	for _, v := range receivers {
		if conn, ok := k_online_users[v]; ok {
			K_Msgs <- Message{receiver: v, msg: data, conn: conn}
		}
	}
}

func (this *ChatController) Broadcast(j *simplejson.Json) {
	data, err := j.MarshalJSON()
	if err != nil {
		logs.Error("SendMsg MarshalJSON failed. Error: ", err.Error())
		panic(err)
	}
	for k, v := range k_online_users {
		K_Msgs <- Message{receiver: k, msg: data, conn: v}
	}
}

// @router / [get]
func (this *ChatController) WSConnect() {

	// Upgrade from http request to WebSocket.
	ws, err := websocket.Upgrade(this.Ctx.ResponseWriter, this.Ctx.Request, nil, WS_READ_BUFFER_SIZE, WS_WRITE_BUFFER_SIZE)
	if _, ok := err.(websocket.HandshakeError); ok {
		http.Error(this.Ctx.ResponseWriter, "Not a websocket handshake", 400)
		return
	} else if err != nil {
		logs.Error("Cannot setup WebSocket connection:", err)
		return
	}
	this.ws = ws
	this.cur_cmd = ""
	this.cur_user = ""
	this.cur_user_type = models.USER_NORMAL_TYPE

	// Message receive loop.
	var err_num = 0
	for {

		defer func() {
			delete(k_online_users, this.cur_user)
			ws.Close()
			this.ws = nil

		}()

		_, body, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, WS_CLOSE_ERROR...) {
				logs.Error("Close Error: %s, cur_user: %s, cur_user_type: %d, conn: %p", err.Error(), this.cur_user, this.cur_user_type, this.ws)
				return
			}
			logs.Error("Read msg faled from WebSocket. Error:", err.Error())
			err_num++
			if err_num >= 20 {
				return
			}
			continue
		}
		err_num = 0

		this.body_json, err = simplejson.NewJson(body)
		if err != nil {
			logs.Error("Request Body is NOT in JSON format. Error:", err.Error())
			this.ErrReply(PARSE_JSON_ERR)
			continue
		}

		this.cur_cmd, err = this._Parse()
		if err != nil {
			logs.Error("Miss parameter \"type\" in Json body. Error:", err.Error())
			this.ErrReply(MISS_PARAM_ERR)
			continue
		}

		switch this.cur_cmd {
		case "login":
			this._Login()

		case "adduser":
			this._AddUser()

		case "deluser":
			this._DeleteUser()

		case "listuser":
			this._ListUser()

		case "sendmsg":
			this._SendMsg()

		default:
			logs.Error("Unknown cmd: ", this.cur_cmd)
			this.ErrReply(CMD_TYPE_ERR)
		}
	}
}

func (this *ChatController) _Parse() (string, error) {
	cmd, err := this.body_json.Get("type").String()
	if err != nil {
		return "", err
	}

	return cmd, nil
}

func (this *ChatController) _Login() {
	name := this.body_json.Get("name").MustString()
	password := this.body_json.Get("password").MustString()

	if id, user_type := models.UserLogin(name, password); id != 0 {
		this.cur_user = name
		this.cur_user_type = user_type
		j := this._ConstructReplyJson()
		if user_type == 0 {
			j.Set("admin", true)
		}
		this.Reply(j)

		// update online conn
		if _, ok := k_online_users[this.cur_user]; ok {
			k_online_users[this.cur_user].Close()
		}
		k_online_users[this.cur_user] = this.ws

		// send welcome msg
		j = this._ConstructMsgJson(_WelcomMsg(name))
		this.Broadcast(j)
	} else {
		this.ErrReply(LOGIN_ERR)
	}
}

func (this *ChatController) _AddUser() {
	if this.cur_user == "" || this.cur_user_type != 0 {
		this.ErrReply(PERMISSION_ERR)
		return
	}

	name := this.body_json.Get("name").MustString()
	password := this.body_json.Get("password").MustString()

	if id := models.AddUser(name, password); id != 0 {
		j := this._ConstructReplyJson()
		this.Reply(j)
	} else {
		this.ErrReply(ADD_USER_ERR)
	}
}

func (this *ChatController) _DeleteUser() {
	if this.cur_user == "" || this.cur_user_type != 0 {
		this.ErrReply(PERMISSION_ERR)
		return
	}

	var users []string = nil
	is_remove_all := this.body_json.Get("removeall").MustBool()
	if !is_remove_all {
		users = this.body_json.Get("users").MustStringArray()
		if len(users) == 0 {
			logs.Error("\"users\" array is empty.")
			this.ErrReply(MISS_PARAM_ERR)
			return
		}
	}

	if models.DeleteUser(users, is_remove_all) {
		j := this._ConstructReplyJson()
		this.Reply(j)
	} else {
		this.ErrReply(DELETE_USER_ERR)
	}
}

func (this *ChatController) _ListUser() {
	if this.cur_user == "" {
		this.ErrReply(PERMISSION_ERR)
		return
	}

	start := this.body_json.Get("start").MustInt()
	length := this.body_json.Get("length").MustInt()

	users := models.ListUser(start, length)
	j := this._ConstructReplyJson()
	j.Set("users", users)
	this.Reply(j)
}

func (this *ChatController) _SendMsg() {
	if this.cur_user == "" {
		this.ErrReply(PERMISSION_ERR)
		return
	}

	receivers := this.body_json.Get("receivers").MustStringArray()
	if len(receivers) == 0 {
		logs.Error("\"receivers\" array is empty.")
		this.ErrReply(MISS_PARAM_ERR)
		return
	}
	msg := this.body_json.Get("msg").MustString()

	j := this._ConstructReplyJson()
	this.Reply(j)

	j = this._ConstructMsgJson(msg)
	this.SendMsg(j, receivers)
}

func (this *ChatController) _ConstructReplyJson() *simplejson.Json {
	j := simplejson.New()
	j.Set("version", 1)
	j.Set("code", 0)
	j.Set("type", this.cur_cmd)

	return j
}

func (this *ChatController) _ConstructMsgJson(msg string) *simplejson.Json {
	j := simplejson.New()
	j.Set("version", 1)
	j.Set("sender", this.cur_user)
	j.Set("type", "recvmsg")
	j.Set("msg", msg)

	//time_unix := time.Now().Format(time.RFC1123Z)
	time_unix := time.Now().Unix()
	j.Set("time", time_unix)

	return j
}

func _WelcomMsg(name string) string {
	return fmt.Sprintf(WELCOMD_MSG, name)
}

func _ByeMsg(name string) string {
	return fmt.Sprintf(BYEBYE_MSG, name)
}
