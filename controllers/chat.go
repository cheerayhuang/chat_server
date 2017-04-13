package controllers

import (
	"fmt"
	"strconv"
	"strings"

	"chat_server/models"

	"github.com/astaxie/beego"
	"github.com/bitly/go-simplejson"
	"github.com/gorilla/websocket"
)

const (
	PARSE_JSON_ERR = 1000
	MISS_PARAM_ERR = 1100
	CMD_TYPE_ERR   = 1200
	PERMISSION_ERR = 1300
	LOGIN_ERR      = 2000

	ERR_REPLYS = map[int]string{
		PARSE_JSON_ERR: "Message is NOT in JSON format.",
		MISS_PARAM_ERR: "Miss parameters in JSON.",
		CMD_TYPE_ERR:   "Unknown command type.",
		LOGIN_ERR:      "Login failed. User does NOT exist or password is Wrong.",
		PERMISSION_ERR: "No user login or the user doesn't have permisson to exec this command.",
	}
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
	j.Set("code", err_code)
	j.Set("reason", ERR_REPLYS[err_code])

	this.Reply(j)
}

func (this *ChatController) Reply(j *simplejson.Json) {
	if this.ws {
		data, err := j.MarshalJSON()
		if err != nil {
			logs.Error("MarshalJSON failed.")
		} else {
			this.ws.WriteMessage(websocket.TextMessage, data)
		}
	} else {
		logs.Error("Current connetion is lost.")
	}
}

// @router / [get]
func (this *ChatController) WSConnect() {

	// Upgrade from http request to WebSocket.
	ws, err := websocket.Upgrade(this.Ctx.ResponseWriter, this.Ctx.Request, nil, 1024, 1024)
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
	this.cur_user_type = 1024

	// Message receive loop.
	for {
		_, body, err := ws.ReadMessage()
		if err != nil {
			logs.Error("Read msg faled from WebSocket. Error:", err.Error())
			continue
		}

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
	if users {
		j.Set("users", users)
	} else {
		j.Set("users", make([]string, 0))
	}

	this.Reply(j)
}

func (this *ChatController) _ConstructReplyJson() *simplejson.Json {
	j := simplejson.New()
	j.Set("version", 1)
	j.Set("code", 0)
	j.Set("type", this.cur_cmd)

	return j
}
