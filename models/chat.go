package models

import (
	"chat_server/models/db"

	"github.com/astaxie/beego/logs"
)

const (
	USER_ROOT_TYPE   = 0
	USER_ADMIN_TYPE  = 1
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

func AddUser(cur_id int64, cur_type int, name, password string) int64 {
	logs.Debug("add user cur_id: %d, cur_type: %d", cur_id, cur_type)
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
		data := map[string]interface{}{
			"user_name":  name,
			"passwd":     password,
			"user_type":  USER_NORMAL_TYPE,
			"created_by": cur_id,
		}
		if cur_type == USER_ROOT_TYPE {
			data["user_type"] = USER_ADMIN_TYPE
		}
		id, err := mysql.Insert(data)
		if err != nil {
			logs.Error("db Insert operation failed. Error: ", err.Error())
			return 0
		}
		return id
	}

	logs.Warning("User already exists.")
	return 0
}

func DeleteUser(cur_id int64, cur_type int, users []string, is_remove_all bool) bool {
	logs.Debug("delete user cur_id: %d, cur_type: %d", cur_id, cur_type)
	logs.Debug("delete user, is_remove_all: ", is_remove_all)
	logs.Debug("delete users: ", users)

	if _, ok := mysql.(*db.DBase); !ok {
		Init()
	}
	mysql.(*db.DBase).SetDefaultTable("chat_users")

	if is_remove_all {
		var err error
		if cur_type == USER_ROOT_TYPE {
			err = mysql.Where("user_name !=", "root").Delete()
		} else {
			err = mysql.Where("created_by", cur_id).Delete()
		}
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

func ListUser(id int64, start, length int) []string {
	logs.Debug("list user cur_id: %d", id)
	logs.Debug("list user, start: ", start)
	logs.Debug("list user, length: ", length)

	if _, ok := mysql.(*db.DBase); !ok {
		Init()
	}
	mysql.(*db.DBase).SetDefaultTable("chat_users")

	users := make([]string, 0)
	rows, err := mysql.Select("user_name").Where("created_by", id).Limit(start, length).Query()
	if err != nil {
		logs.Error("db Query operation failed. Error: ", err.Error())
		return users
	}
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
