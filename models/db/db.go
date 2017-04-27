package db

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"fmt"
	"shiftred/error"
	"strconv"
	"strings"

	"github.com/astaxie/beego/logs"
)

type DB interface {
	Query(stat *DBStat) (*sql.Rows, error)
	Insert(values map[string]interface{}, stat *DBStat) (int64, error)
	Delete(stat *DBStat) error
	Update(values map[string]interface{}, stat *DBStat) error
	Count(stat *DBStat) (int64, error)
	Exist(stat *DBStat) (bool, error)
	Close() error
}

type DBase struct {
	d        string
	user     string
	pwd      string
	database string
	host     string
	port     string
	table    string
	db       *sql.DB
}

type DBStat struct {
	table string

	d_stat string
	q_stat string
	u_stat string
}

func New(db, user, pwd, database, host, port string) (DB, error) {
	if db == "" || user == "" || pwd == "" || database == "" || host == "" || port == "" {
		return nil, MyErr.New(MyErr.DB_CONN_MISS_PARAMS, "miss Database Connection Paramters")
	}

	r := new(DBase)
	r.user = user
	r.pwd = pwd
	r.database = database
	r.host = host
	r.port = port
	r.d = db

	var err error
	if db == "postgres" {
		conn_str := []string{"user=" + r.user, "password=" + r.pwd, "dbname=" + r.database, "host=" + r.host, "port=" + r.port, "sslmode=disable"}
		r.db, err = sql.Open(db, strings.Join(conn_str, " "))
		if err != nil {
			return nil, err
		}
	} else {
		conn_str := r.user + ":" + r.pwd + "@tcp(" + r.host + ":" + r.port + ")/" + database
		r.db, err = sql.Open(db, conn_str)
		if err != nil {
			return nil, err
		}
	}

	return r, nil
}

func NewDBStat(table string) (*DBStat, error) {
	if table == "" {
		return nil, fmt.Errorf("Create DBStat failed, table name is an empty string.")
	}

	r := new(DBStat)
	r.table = table
	r.q_stat = "SELECT * FROM [table]"
	r.d_stat = "DELETE FROM [table]"
	r.u_stat = "UPDATE [table] SET [vars]"

	return r, nil
}

func (this *DBStat) SetTable(name string) {
	if name == "" {
		return
	}

	this.table = name
}

func (this *DBase) Query(stat *DBStat) (*sql.Rows, error) {
	logs.Debug("DB Query Sql: ", stat.q_stat)

	defer stat.ResetStat()
	rows, err := this.db.Query(stat.q_stat)
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (this *DBase) Count(stat *DBStat) (int64, error) {
	var count int64

	stat.q_stat = strings.Replace(stat.q_stat, "*", "COUNT(*)", 1)
	logs.Debug("DB Count Sql: ", stat.q_stat)

	defer stat.ResetStat()
	err := this.db.QueryRow(stat.q_stat).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (this *DBase) Exist(stat *DBStat) (bool, error) {
	count, err := this.Count(stat)
	if err != nil {
		return false, err
	}
	if count != 0 {
		return true, nil
	}

	return false, nil
}

func (this *DBase) Delete(stat *DBStat) error {
	logs.Debug("DB Delete Sql: ", stat.d_stat)

	defer stat.ResetStat()
	stmt, err := this.db.Prepare(stat.d_stat)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	return nil
}

func (this *DBase) Update(values map[string]interface{}, stat *DBStat) error {
	var set_st string

	for k, v := range values {
		set_st += (k + " = ")
		switch val := v.(type) {
		case string:
			set_st += "'" + val + "'"
		case int:
			set_st += strconv.FormatInt(int64(val), 10)
		case int64:
			set_st += strconv.FormatInt(int64(val), 10)
		case float32:
			set_st += strconv.FormatFloat(float64(val), 'f', -1, 64)
		case float64:
			set_st += strconv.FormatFloat(float64(val), 'f', -1, 64)
		}
		set_st += ", "
	}
	set_st = strings.TrimRight(set_st, ", ")

	stat.u_stat = strings.Replace(stat.u_stat, "[vars]", set_st, 1)
	logs.Debug("DB Update Sql: ", stat.u_stat)

	defer stat.ResetStat()
	stmt, err := this.db.Prepare(stat.u_stat)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	return nil
}

func (this *DBase) Insert(values map[string]interface{}, stat *DBStat) (int64, error) {
	l := len(values)
	if l == 0 {
		return 0, MyErr.New(MyErr.DB_INSERT_MISS_VALUES, "miss values in insert statement.")
	}

	var marks []string
	var keys []string
	var vals []interface{}
	var i int = 0
	for k, v := range values {
		i++
		if this.d == "postgres" {
			marks = append(marks, "$"+strconv.Itoa(i))
		} else {
			marks = append(marks, "?")
		}
		keys = append(keys, k)
		vals = append(vals, v)
	}
	marks_str := strings.Join(marks, ",")
	keys_str := strings.Join(keys, ",")
	keys_str = "(" + keys_str + ")"
	stmt_str := "INSERT INTO " + stat.table + keys_str + " VALUES(" + marks_str + ")"
	logs.Debug("DB Insert Sql: ", stmt_str)
	stmt, err := this.db.Prepare(stmt_str)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(vals...)
	if err != nil {
		return 0, err
	}

	row_count, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return row_count, nil
}

// it is rarely necessary to call it,
// as golang demand sql driver should implemnt a connections pool for Database.
func (this *DBase) Close() error {
	return this.db.Close()
}

func (this *DBStat) Select(fields ...string) *DBStat {
	var fields_st = strings.Join(fields, ",")
	this.q_stat = strings.Replace(this.q_stat, "*", fields_st, 1)

	return this
}

func (this *DBStat) Where(field string, value interface{}) *DBStat {
	return this._Where(field, value, false)
}

func (this *DBStat) OrWhere(field string, value interface{}) *DBStat {
	return this._Where(field, value, true)
}

func (this *DBStat) _Where(field string, value interface{}, or bool) *DBStat {
	is_first := false
	if !strings.Contains(this.q_stat, "WHERE") {
		is_first = true
	}

	var where_st string
	if is_first {
		where_st += " WHERE " + field
	} else {
		if !or {
			where_st += " AND " + field
		} else {
			where_st += " OR " + field
		}
	}

	if _, ok := _GetWhereOperator(field); !ok {
		where_st += " = "
	}

	switch v := value.(type) {
	case string:
		where_st += "'" + v + "'"
	case int:
		where_st += strconv.FormatInt(int64(v), 10)
	case int64:
		where_st += strconv.FormatInt(int64(v), 10)
	case float32:
		where_st += strconv.FormatFloat(float64(v), 'f', -1, 64)
	case float64:
		where_st += strconv.FormatFloat(float64(v), 'f', -1, 64)
	}

	this.q_stat += where_st
	this.d_stat += where_st
	this.u_stat += where_st

	return this
}

func (this *DBStat) From() *DBStat {
	this.q_stat = strings.Replace(this.q_stat, "[table]", this.table, 1)
	this.d_stat = strings.Replace(this.d_stat, "[table]", this.table, 1)
	this.u_stat = strings.Replace(this.u_stat, "[table]", this.table, 1)

	return this
}

func (this *DBStat) Limit(start, length int) *DBStat {
	this.q_stat += " LIMIT " + strconv.FormatInt(int64(start), 10) + ", " + strconv.FormatInt(int64(length), 10)

	return this
}

func (this *DBStat) ResetStat() {
	this.q_stat = "SELECT * FROM [table]"
	this.d_stat = "DELETE FROM [table]"
	this.u_stat = "UPDATE [table] SET [vars]"
}

func _GetWhereOperator(field string) (string, bool) {
	if strings.Contains(field, ">=") {
		return ">=", true
	}

	if strings.Contains(field, "<=") {
		return "<=", true
	}

	if strings.Contains(field, "!=") {
		return "!=", true
	}

	if strings.Contains(field, "=") {
		return "=", true
	}

	if strings.Contains(field, ">") {
		return ">", true
	}

	if strings.Contains(field, ">") {
		return "<", true
	}

	return "", false
}
