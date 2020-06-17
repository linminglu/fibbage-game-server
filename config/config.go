package config

import (
	"github.com/jinzhu/configor"
)

type (
	Logger struct {
		LogsFileEnabled    bool
		LogFilePath        string
		LogFileName        string
		LogRotateMegabytes int
		LogRotateDuration  int
		LogRotateFiles     int
		LogDebugMode       bool
	}
	Database struct {
		Driver   string `yaml:"driver"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Name     string `yaml:"name"`
		Server   string `yaml:"server"`
		Port     int    `yaml:"port"`
	}
)

var Configuration = struct {
	Port     uint `default:"7000" env:"port"`
	Logger   Logger
	Database Database
}{}

func init() {
	if err := configor.Load(&Configuration, "config.yml"); err != nil {
		panic(err)
	}
}
