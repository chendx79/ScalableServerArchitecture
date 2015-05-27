package CryNet

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/ziutek/mymysql/mysql"
	_ "github.com/ziutek/mymysql/thrsafe"
	"log"
)

var db *sql.DB
var conn mysql.Conn
var ignore_list = []string{"status"}

func InitMySQL() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", DBUser, DBPass, DBHost, DBPort, DBName)
	db, _ = sql.Open("mysql", dsn)

	conn = mysql.New("tcp", "", fmt.Sprintf("%s:%d", DBHost, DBPort), DBUser, DBPass, DBName)
	err := conn.Connect()
	if err != nil {
		panic(err)
	}
}

func ReconnectMySQL() {
	CloseMySQL()
	InitMySQL()
}

func RunSQL(format string, a ...interface{}) ([]map[string]interface{}, bool) {
	sql := fmt.Sprintf(format, a...)
	//log.Println(sql)
	rows, err := db.Query(sql)
	if err != nil {
		log.Println("query error!!%v\n", err)
		return nil, false
	}
	if rows == nil {
		log.Println("Rows is nil")
		return nil, false
	}
	columns, _ := rows.Columns()
	var result []map[string]interface{}
	for rows.Next() {
		record := make(map[string]interface{})
		valueArray := make([]interface{}, len(columns))
		pointArray := make([]interface{}, len(columns))
		for i := 0; i < len(columns); i++ {
			pointArray[i] = &valueArray[i]
		}
		row_err := rows.Scan(pointArray...)
		if row_err != nil {
			log.Println("Row error!!")
			return nil, false
		}
		for i := 0; i < len(columns); i++ {
			if valueArray[i] == nil {
				record[columns[i]] = ""
			} else {
				if columns[i][len(columns[i])-1:] == "s" {
					listValue, _ := StrToArray(string(valueArray[i].([]byte)))
					record[columns[i]] = listValue
				} else {
					record[columns[i]] = string(valueArray[i].([]byte))
				}
			}
		}
		result = append(result, record)
	}
	return result, true
}

func CallSP(format string, a ...interface{}) ([]map[string]interface{}, bool) {
	sql := fmt.Sprintf(format, a...)
	//log.Println(sql)
	res, res_err := conn.Start(sql)
	if res_err != nil {
		log.Println("Stored procedure run error!", res_err)
		return nil, false
	}
	var result []map[string]interface{}
	for !res.StatusOnly() {
		rows, row_err := res.GetRows()
		if row_err != nil {
			//log.Println("res.GetRows, row error!", row_err)
			break
		}
		if len(rows) == 0 {
			log.Println("no row got!")
		} else {
			fields := res.Fields()

			for _, row := range rows {
				record := make(map[string]interface{})
				for i := 0; i < len(fields); i++ {
					if fields[i].Name[len(fields[i].Name)-1:] == "s" && !stringInSlice(fields[i].Name, ignore_list) {
						listValue, _ := StrToArray(string(row[i].([]byte)))
						record[fields[i].Name] = listValue
					} else {
						record[fields[i].Name] = string(row[i].([]byte))
					}
				}
				result = append(result, record)
			}
		}
		res, err := res.NextResult()
		if err != nil {
			log.Println("res.NextResult, row error!", err)
			break
		}
		if res == nil {
			panic("nil result from procedure")
		}
	}
	return result, true
}

func CloseMySQL() {
	db.Close()
	conn.Close()
}

func test() {
	InitMySQL()
	records, success := CallSP("call ...(%s);", "44")
	log.Println(records, success)
	defer CloseMySQL()
}
