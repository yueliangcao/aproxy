package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Config struct {
	LnAddr string `json:lnAddr`
}

func Parse(path string) (cfg *Config, err error) {
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

	cfg = &Config{}

	if err = json.Unmarshal(data, cfg); err != nil {
		fmt.Println("unmarshal json err :" + err.Error())
		return
	}

	return
}
