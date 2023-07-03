package config

import (
	"encoding/json"
	"io/ioutil"
	"mjump/app/logger"
)

type Config struct {
	Users  []User `json:"users"`
	Hosts  []Host `json:"hosts"`
	Listen Listen `json:"listen"`
}

type Listen struct {
	Host string `json:"host"`
	Port string `json:"port"`
}

type Host struct {
	ID         string `json:"id"`   // ID 是唯一的
	Name       string `json:"name"` //Name 是唯一的
	Username   string `json:"username"`
	IP         string `json:"ip"`
	Password   string `json:"password"`
	Prot       string `json:"prot"`
	Isactive   string `json:"isactive"`
	Ftpdir     string `json:"ftpdir"`
	PrivateKey string `json:"privateKey"`
}

type User struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	Public   string   `json:"public"`
	Assets   []string `json:"assets"`
}

func Loadcnf() (Config, error) {
	var conf Config
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		logger.Error("读取Json 文件错误.", err)
		return conf, err
	}
	err = json.Unmarshal(data, &conf)
	if err != nil {
		logger.Error("解析Json 文件错误.", err)
		return conf, err
	}
	return conf, nil
}
