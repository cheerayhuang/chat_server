package db

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"shiftred/error"
	"strconv"
	"strings"

	"github.com/astaxie/beego/logs"
)

type DB interface {
	// statement
	Select(fields ...string)
	Where(field string, value interface{})
	//OrWhere(field string, value interface{})
	From(table string)
	Limit(start, length int)

	// operation
	Query() (*sql.Rows, error)
	Insert(...interface{}) (int64, error)
	Delete() error
	Update()
	Count() (int64, error)
	Exist() (bool, error)
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
	fields   []string
	db       *sql.DB
	tm_start int64
	tm_end   int64

	q_stat string
	d_stat string
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

	r.q_stat = "SELECT * FROM [table]"
	r.d_stat = "DELETE FROM [table]"

	return r, nil
}

func (this *DBase) SetDefaultTable(name string) {
	if name == "" {
		return
	}

	this.table = name
}

func (this *DBase) Query() (*sql.Rows, error) {
	if stirngs.Contains(this.q_stat, "[table]") {
		this.q_stat = strings.Replace(this.q_stat, "[table]", this.table, 1)
	}

	rows, err := this.db.Query(this.q_stat)
	if err != nil {
		return nil, err
	}

	this._ResetStat()

	return rows, nil
}

func (this *DBase) Count() (int64, error) {
	var count int64

	if stirngs.Contains(this.q_stat, "[table]") {
		this.q_stat = strings.Replace(this.q_stat, "[table]", this.table, 1)
	}
	this.q_stat = strings.Replace(this.q_stat, "*", "COUNT(*)", 1)

	err := this.db.QueryRow(this.q_stat).Scan(&count)
	if err != nil {
		return 0, err
	}

	this._ResetStat()

	return count, nil
}

func (this *DBase) Exist() (bool, error) {
	count, err := this.Count()
	if err != nil {
		return false, err
	}
	if count != 0 {
		return true, nil
	}

	return false, nil
}

func (this *DBase) Delete() error {
	if stirngs.Contains(this.d_stat, "[table]") {
		this.d_stat = strings.Replace(this.d_stat, "[table]", this.table, 1)
	}
	logs.Debug(this.d_stat)

	stmt, err := this.db.Prepare(this.d_stat)
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

func (this *DBase) Update() {

}

func (this *DBase) Insert(args ...interface{}) (int64, error) {
	l := len(args)
	if l == 0 {
		return 0, MyErr.New(MyErr.DB_INSERT_MISS_VALUES, "miss values in insert statement.")
	}

	var marks []string
	for i := 1; i <= l; i++ {
		if this.d == "postgres" {
			marks = append(marks, "$"+strconv.Itoa(i))
		} else {
			marks = append(marks, "?")
		}
	}
	marks_str := strings.Join(marks, ",")
	stmt_str := "INSERT INTO " + this.table + " VALUES(" + marks_str + ")"
	//logs.Debug(stmt_str)
	stmt, err := this.db.Prepare(stmt_str)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	res, err := stmt.Exec(args...)
	if err != nil {
		return 0, err
	}

	row_count, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return row_count, nil
}

func (this *DBase) Close() error {
	return this.db.Close()
}

func (this *DBase) Select(fields ...string) *DBase {
	var fields_st = strings.Join(fields, ",")
	this.q_stat = strings.Replace(this.q_stat, "*", fields_st, 1)

	return this
}

func (this *DBase) Where(field string, value interface{}) *Dbase {
	is_first := false
	if !strings.Contains(this.q_stat, "WHERE") {
		is_first = true
	}

	var where_st string
	if is_first {
		where_st += " WHERE `" + field + "` = "
	} else {
		where_st += " AND `" + field + "` = "
	}

	switch v := value.(type) {
	case string:
		where_st += "'" + v + "'"
	case int:
	case int64:
		where_st += strconv.FormatInt(int64(v), 10)
	}

	this.q_stat += where_st
	this.d_stat += where_st

	return this
}

func (this *DBase) From(table string) *DBase {
	this.q_stat = strings.Replace(this.q_stat, "[table]", table, 1)
	this.d_stat = strings.Replace(this.d_stat, "[table]", table, 1)

	return this
}

func (this *DBase) Limit(start, length int) *DBase {
	this.q_stat += " LIMIT " + strconv.FormatInt(int64(start), 10) + ", " + strconv.FormatInt(int64(length), 10)

	return this
}

func (this *DBase) _ResetStat() {
	this.q_stat = "SELECT * FROM [table]"
	this.d_stat = "DELETE FROM [table]"
}
