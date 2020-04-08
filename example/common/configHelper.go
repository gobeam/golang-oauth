package common

import (
	"github.com/go-ini/ini"
	"os"
)

var configPath string

func init() {
	path, _ := os.Getwd()
	SetConfigPath(path + "/config/config.ini")
}

func SetConfigPath(path string) {
	configPath = path
}

func GetConfig(section string, key string) *ini.Key {
	cfg, _ := ini.InsensitiveLoad(configPath)
	v, _ := cfg.Section(section).GetKey(key)

	return v
}
