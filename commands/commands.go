package commands

import (
	"encoding/json"
	"io/ioutil"
)

func New() {
	filename := "./redis-commands.json"
	plan, _ := ioutil.ReadFile(filename)
	var data interface{}
	err := json.Unmarshal(plan, &data)
	if nil != err {
		panic("Unable to load commands")
	}
}
