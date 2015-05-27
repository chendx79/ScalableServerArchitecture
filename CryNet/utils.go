package CryNet

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/nu7hatch/gouuid"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var ErrStringNotArray = errors.New("The string is not array")

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func Now() string {
	now := time.Now()
	nowString := now.Format("2006-01-02 15:04:05")
	return nowString
}

func Today() string {
	now := time.Now()
	todayString := now.Format("2006-01-02")
	return todayString
}

func NowString() string {
	now := time.Now()
	millisecond := now.Nanosecond() / 1000000
	nowString := now.Format("20060102150405")
	nowString = fmt.Sprintf("%s%d", nowString, millisecond)
	return nowString
}

func GetCurrentPath() string {
	fpath, _ := exec.LookPath(os.Args[0])
	dir, _ := path.Split(fpath)
	os.Chdir(dir)
	wd, _ := os.Getwd()
	path := fmt.Sprintf("%s/", wd)
	return path
}

func GetExt(filename string) string {
	ext := filepath.Ext(filename)
	ext = string(ext[1:])
	return ext
}

func GetUUID() string {
	u, _ := uuid.NewV4()
	return strings.Replace(u.String(), "-", "", 4)
}

func ReadFile(path string) string {
	fi, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fi.Close()
	fd, err := ioutil.ReadAll(fi)
	return string(fd)
}

func IntToStr(i int) string {
	return strconv.Itoa(i)
}

func StrToInt(s string) int {
	result, err := strconv.Atoi(s)
	if err != nil {
		log.Panicln(err)
	}
	return result
}

func MapStrToInt(data map[string]interface{}, columns ...string) map[string]interface{} {
	for _, column := range columns {
		data[column] = StrToInt(data[column].(string))
	}
	return data
}

func ArrayStrToInt(array []map[string]interface{}, columns ...string) []map[string]interface{} {
	for _, data := range array {
		for _, column := range columns {
			data[column] = StrToInt(data[column].(string))
		}
	}

	return array
}

/*
   [[map[id:16 count:1] map[id:17 count:2]] => map[id:[16,17], count:[1,2]]
*/
func Data2ValueList(data []interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	length := len(data)
	for k, _ := range data[0].(map[string]interface{}) {
		result[k] = make([]interface{}, length)
	}
	for i := 0; i < len(data); i++ {
		for k, _ := range result {
			result[k].([]interface{})[i] = data[i].(map[string]interface{})[k]
		}
	}
	return result
}

func StrToArray(str string) ([]interface{}, error) {
	if str == "" {
		return []interface{}{}, nil
	}
	//['B', ['T', 'T', 'T', 'T', 'T']]
	newStr := strings.Replace(str, "['", `["`, -1)
	newStr = strings.Replace(newStr, "']", `"]`, -1)
	newStr = strings.Replace(newStr, "u'", `"`, -1)
	newStr = strings.Replace(newStr, ", '", `, "`, -1)
	newStr = strings.Replace(newStr, "',", `",`, -1)
	newStr = strings.Replace(newStr, `u"`, `"`, -1)
	//log.Println("StrToArray", str, newStr)
	js, err := simplejson.NewJson([]byte(newStr))
	result, _ := js.Array()
	//log.Println("StrToArray", str, newStr, js, result)
	return result, err
}

func ArrayToJsonString(array []interface{}) string {
	jsonString, _ := json.Marshal(array)
	return string(jsonString)
}

func FileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil || os.IsExist(err)
}

func ReformatRecord(data map[string]interface{}) map[string]interface{} {
	for k, v := range data {
		if v == "" {
			continue
		}
		if k[len(k)-1:] == "s" && !stringInSlice(k, ignore_list) {
			//log.Println(k, v)
			data[k], _ = StrToArray(v.(string))
		}
	}
	return data
}
