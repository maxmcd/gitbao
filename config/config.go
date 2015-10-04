package config

import (
	"encoding/json"
	"io/ioutil"
)

var C map[string]string

func init() {
	config, err := ioutil.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(config, &C)
	if err != nil {
		panic(err)
	}
}
