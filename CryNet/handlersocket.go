package CryNet

import (
	"errors"
	"fmt"
	"github.com/bketelsen/handlersocket-go"
	"log"
	"strings"
)

var indexCache map[string]int
var indexConnections map[int]*handlersocket.HandlerSocket
var currentIndexId int
var ErrNoRecords = errors.New("No records")

func InitHandlerSocket() {
	indexCache = make(map[string]int)
	indexConnections = make(map[int]*handlersocket.HandlerSocket)
	currentIndexId = 1
}

func getIndexIdConnection(tableName string, columns []string, indexName string) (int, *handlersocket.HandlerSocket, error) {
	columnsStr := strings.Join(columns, ",")
	cacheKey := fmt.Sprintf("%s:%s:%s:%s", DBName, tableName, columnsStr, indexName)
	indexId, exists := indexCache[cacheKey]
	var hs *handlersocket.HandlerSocket
	if !exists {
		indexId = currentIndexId

		indexCache[cacheKey] = indexId
		//log.Println("indexCache[cacheKey] = indexId", cacheKey, indexId)
		currentIndexId = currentIndexId + 1

		hs = handlersocket.New()
		connectErr := hs.Connect(DBHost, HandlerSocketReadPort, HandlerSocketWritePort)
		if connectErr != nil {
			log.Panicln("Handler socket of MySQL connecting failed")
			return 0, nil, connectErr
		}
		indexConnections[indexId] = hs
	} else {
		hs, exists = indexConnections[indexId]
		if !exists {
			log.Println("hs does not exist", cacheKey, indexCache, indexConnections, indexId)
			return 0, nil, errors.New("hs does not exist")
		}

		//lock, exists = indexLocks[indexId]
		if !exists {
			log.Println("lock does not exist", cacheKey, indexCache, indexConnections, indexId)
			return 0, nil, errors.New("lock does not exist")
		}
	}
	openErr := hs.OpenIndex(indexId, DBName, tableName, indexName, columns...)
	if openErr != nil {
		log.Panicln("hs.OpenIndex error,", openErr, indexId, DBName, tableName, indexName, columnsStr)
		return 0, nil, openErr
	}
	return indexId, hs, nil
}

func FindSingle(tableName string, indexName string, oper string, compareValues []string, outputColumns []string, limit int, offset int) (map[string]interface{}, error) {
	indexId, hs, _ := getIndexIdConnection(tableName, outputColumns, indexName)
	records, err := hs.Find(indexId, oper, limit, offset, compareValues...)
	if err != nil {
		log.Panicln("hs.Find error,", records, err)
	}
	if len(records) == 0 {
		log.Println("hs.Find error, No records")
		return nil, ErrNoRecords
	}

	return records[0].Data, err
}

func FindMulti(tableName string, indexName string, oper string, compareValues []string, outputColumns []string, limit int, offset int) ([]interface{}, error) {
	indexId, hs, _ := getIndexIdConnection(tableName, outputColumns, indexName)
	records, err := hs.Find(indexId, oper, limit, offset, compareValues...)
	if err != nil {
		log.Panicln("hs.Find error,", records, err)
	}
	if len(records) == 0 {
		log.Println("hs.Find error, No records")
		return nil, ErrNoRecords
	}
	result := make([]interface{}, len(records))
	for i := 0; i < len(records); i++ {
		result[i] = records[i].Data
	}

	return result, err
}

func FindIds(tableName string, indexName string, oper string, compareValues []string, outputColumn string, limit int, offset int) ([]string, error) {
	outputColumns := []string{outputColumn}
	indexId, hs, _ := getIndexIdConnection(tableName, outputColumns, indexName)
	records, err := hs.Find(indexId, oper, limit, offset, compareValues...)
	if err != nil {
		log.Panicln("hs.Find error,", records, err)
	}
	if len(records) == 0 {
		log.Println("hs.Find error, No records")
		return nil, ErrNoRecords
	}
	result := make([]string, len(records))
	for i := 0; i < len(records); i++ {
		log.Println(records[i].Data)
		result[i] = records[i].Data[outputColumn].(string)
	}

	return result, err
}

func FindAll(tableName string, outputColumns []string) ([]interface{}, error) {
	indexId, hs, _ := getIndexIdConnection(tableName, outputColumns, "PRIMARY")
	records, err := hs.Find(indexId, ">", 10000, 0, "0")
	if err != nil {
		log.Panicln("hs.Find error,", records, err)
	}
	if len(records) == 0 {
		log.Println("hs.Find error, No records")
		return nil, ErrNoRecords
	}

	result := make([]interface{}, len(records))
	for i := 0; i < len(records); i++ {
		result[i] = records[i].Data
	}

	return result, err
}

func Insert(tableName string, indexName string, insertKeyValues map[string]interface{}) error {
	insertColumns := []string{}
	insertValues := []string{}
	for key, value := range insertKeyValues {
		insertColumns = append(insertColumns, key)
		if valueStr, ok := value.(string); ok {
			insertValues = append(insertValues, fmt.Sprintf("%v", strings.TrimRight(valueStr, "\r\n")))
		} else {
			insertValues = append(insertValues, fmt.Sprintf("%v", value))
		}

	}
	indexId, hs, _ := getIndexIdConnection(tableName, insertColumns, indexName)
	err := hs.Insert(indexId, insertValues...)
	if err != nil {
		log.Panicln("hs.Insert error,", err)
	}
	return err
}

func Update(tableName string, indexName string, compareValues []string, updateKeyValues map[string]interface{}, limit int, offset int) error {
	updateColumns := []string{}
	updateValues := []string{}
	for key, value := range updateKeyValues {
		updateColumns = append(updateColumns, key)
		updateValues = append(updateValues, fmt.Sprintf("%v", strings.TrimRight(value.(string), "\r\n")))
	}
	indexId, hs, _ := getIndexIdConnection(tableName, updateColumns, indexName)
	_, err := hs.Modify(indexId, "=", limit, offset, "U", compareValues, updateValues)
	if err != nil {
		log.Panicln("hs.Modify error,", err)
	}
	return err
}

func Delete(tableName string, indexName string, oper string, compareValues []string, outputColumns []string, limit int, offset int) (map[string]interface{}, error) {
	indexId, hs, _ := getIndexIdConnection(tableName, outputColumns, indexName)
	records, err := hs.Find(indexId, oper, limit, offset, compareValues...)

	if err != nil {
		log.Panicln("hs.Find error,", records, err)
	}
	if len(records) == 0 {
		log.Println("hs.Find error, No records")
		return nil, ErrNoRecords
	}

	_, err = hs.Modify(indexId, "=", limit, offset, "D", compareValues, nil)
	if err != nil {
		log.Panicln("hs.Delete error,", err)
	}
	return records[0].Data, err
}

func CloseHandlerSocket() {
	for _, conn := range indexConnections {
		conn.Close()
	}
}
