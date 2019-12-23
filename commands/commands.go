package commands

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

func New() {
	filename := "redis-commands.json"
	plan, _ := ioutil.ReadFile(filename)
	var data interface{}
	err := json.Unmarshal(plan, &data)
	if nil != err {
		fmt.Println(err)
		panic("Unable to load commands")
	}
	fmt.Println(data.(map[string]interface{})["keys"])
}

func checkType(data interface{}) (res interface{}) {
	switch data.(type) {
	case bool:
		res = data
	}
	// bool, for JSON booleans
	// float64, for JSON numbers
	// string, for JSON strings
	// []interface{}, for JSON arrays
	// map[string]interface{}, for JSON objects
	// nil for JSON null
}

// func main() {
// 	New()
// }
