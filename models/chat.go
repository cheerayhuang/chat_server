package models

import (
	"chat_server/models/db"

	"database/sql"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/astaxie/beego/logs"
	"github.com/bitly/go-simplejson"
)

const (
	USER_ADMIN_TYPE  = 0
	USER_NORMAL_TYPE = 1024
)

var mysql db.DB

func Init() {
	var err error
	mysql, err = db.New("mysql",
		"sdkbox",
		"1234",
		"chat",
		"localhost",
		"3306",
	)
	if err != nil {
		logs.Error("create mysql object failed: ", err.Error())
	}
}

func UserLogin(name, password string) (int64, int) {

	logs.Debug("login name: ", name)
	logs.Debug("password: ", password)

	if _, ok := mysql.(*db.DBase); !ok {
		Init()
	}
	mysql.(*db.DBase).SetDefaultTable("chat_users")

	is_exist, err := mysql.Where("user_name", name).Exist()
	if err != nil {
		logs.Error("db Exist operation failed. Error: ", err.Error())
		return 0, 0
	}

	if is_exist {
		rows, err := mysql.Select("id", "user_name", "passwd", "user_type").Where("user_name", name).Query()
		if err != nil {
			logs.Error("db Query operation failed. Error: ", err.Error())
			return 0, 0
		}
		defer rows.Close()
		var (
			id        int64
			user_name string
			passwd    string
			user_type int
		)
		for rows.Next() {
			err := rows.Scan(&id, &user_name, &passwd, &user_type)
			if err != nil {
				logs.Error("db Rows Scan operation failed. Error: ", err.Error())
				return 0, 0
			}
		}
		if password != passwd {
			logs.Warning("User password is Wrong.")
			return 0, 0
		}
		return id, user_type
	}

	logs.Warning("User does NOT exist.")
	return 0, 0
}

func AddUser(name, password string) int64 {
	logs.Debug("add user name: ", name)
	logs.Debug("add user passwd: ", password)

	if _, ok := mysql.(*db.DBase); !ok {
		Init()
	}
	mysql.(*db.DBase).SetDefaultTable("chat_users")

	is_exist, err := mysql.Where("user_name", name).Exist()
	if err != nil {
		logs.Error("db Exist operation failed. Error: ", err.Error())
		return 0
	}

	if !is_exist {
		id, err := mysql.Insert(name, password, USER_NORMAL_TYPE)
		if err != nil {
			logs.Error("db Insert operation failed. Error: ", err.Error())
			return 0
		}
		return id
	}

	logs.Warning("User already exists.")
	return 0
}

func DeleteUser(users []string, is_remove_all bool) bool {
	logs.Debug("delete user, is_remove_all: ", is_remove_all)
	logs.Debug("delete users: ", users)

	if _, ok := mysql.(*db.DBase); !ok {
		Init()
	}
	mysql.(*db.DBase).SetDefaultTable("chat_users")

	if is_remove_all {
		err := mysql.Delete()
		if err != nil {
			logs.Error("db Delete operation failed. Error: ", err.Error())
			return false
		}
		return true
	}

	for _, v := range users {
		err := mysql.Where("user_name", v).Delete()
		if err != nil {
			logs.Error("db Delete operation failed. Error: ", err.Error())
			return false
		}
	}

	return true
}

func ListUser(start, length int) []string {
	logs.Debug("list user, start: ", start)
	logs.Debug("list user, length: ", length)

	if _, ok := mysql.(*db.DBase); !ok {
		Init()
	}
	mysql.(*db.DBase).SetDefaultTable("chat_users")

	users := make([]string, 0)
	rows, err := mysql.Select("user_name").Limit(start, length).Query()
	defer rows.Close()
	for rows.Next() {
		var user string
		err := rows.Scan(&user)
		if err != nil {
			logs.Error("db Rows Scan operation failed. Error: ", err.Error())
			return users
		}
		users = append(users, user)
	}

	return users
}
