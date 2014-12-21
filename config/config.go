package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

func Parse(path string) (cfg map[string]interface{}, err error) {
	file, err := os.Open(path)

	if err != nil {
		fmt.Println("open config.json err: " + err.Error())
		return
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("read config.json err: " + err.Error())
		return
	}

	cfg = map[string]interface{}{}

	//fmt.Println(string(data))

	if err = json.Unmarshal(data, &cfg); err != nil {
		fmt.Println("unmarshal json err :" + err.Error())
		return
	}

	return
}
